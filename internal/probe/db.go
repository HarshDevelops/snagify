// Package probe performs active connectivity probes (TCP, DNS, HTTP, TLS,
// and unauthenticated database wire-protocol probes) and captures proxy env
// presence. All probes are read-only: no credentials are sent, HTTP response
// bodies and database payload contents are never read or stored, and the
// only writes to a database socket are the protocol pings required to elicit
// an authentication handshake. No password, key, or query is transmitted.
package probe

// The DB probes defined in this file (Redis PING, Postgres SSLRequest, MySQL
// handshake read) all stay within that boundary.

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

// pgSSLRequest is the 8-byte SSLRequest packet a Postgres client sends to ask
// whether the server is willing to upgrade the connection to TLS. The server
// replies with a single byte "S" (accept) or "N" (reject). Either reply
// proves the server speaks the Postgres wire protocol on the port. No
// StartupMessage, no credentials, no user database name.
//
// Reference: postgres/docs/protocol/message-formats.html#sslrequestnegotiation
var pgSSLRequest = []byte{0, 0, 0, 8, 0x04, 0xd2, 0x16, 0x2f}

// mysqlHandshakeProtocolVersion is the protocol byte (0x0a) in a MySQL
// server's Initial Handshake packet. We read the packet (sufficient to prove
// the server is a MySQL daemon) and verify the version byte; we never send
// a HandshakeResponse, so no credentials are involved.
const mysqlHandshakeProtocolVersion = 0x0a

// maxDBRead is the upper bound on bytes read from a database server when
// sniffing a protocol banner. Plenty for the read-only probes below; we
// always stop sooner if the server speaks less.
const maxDBRead = 512

// runDBProbes is called from Run() and dispatches each configured database
// service to the right per-backend probe.
func runDBProbes(cfg config.Config, defaultTimeout time.Duration) []model.DBProbeResult {
	var out []model.DBProbeResult
	for _, pg := range cfg.Databases.Postgres {
		out = append(out, probePostgres(pg, defaultTimeout))
	}
	for _, my := range cfg.Databases.MySQL {
		out = append(out, probeMySQL(my, defaultTimeout))
	}
	for _, rd := range cfg.Databases.Redis {
		out = append(out, probeRedis(rd, defaultTimeout))
	}
	return out
}

// dial reads from r with timeout using SetReadDeadline; the byte slice
// returned is what the server actually said (often empty or partial when the
// server is unresponsive). Errors are surfaced to the caller.
func readWith(conn net.Conn, n int, timeout time.Duration) ([]byte, error) {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	buf := make([]byte, n)
	total := 0
	for total < n {
		read, err := conn.Read(buf[total:])
		total += read
		if err != nil {
			if total > 0 {
				return buf[:total], err
			}
			return nil, err
		}
	}
	return buf[:total], nil
}

// resolveTimeout picks the per-spec timeout or the default.
func resolveTimeout(over int, fallback time.Duration) time.Duration {
	if over > 0 {
		return time.Duration(over) * time.Millisecond
	}
	return fallback
}

// probePostgres dials the server, sends an SSLRequest, and reads the
// single-byte reply. Both "S" (accept) and "N" (decline) prove the server
// is a Postgres daemon; we treat both as HandshakeOK but report Reachable
// only when we got a clean read.
func probePostgres(pg config.PostgresService, defaultTimeout time.Duration) model.DBProbeResult {
	timeout := resolveTimeout(pg.TimeoutMS, defaultTimeout)
	host, port := pg.Host, pg.Port
	if host == "" {
		host = "localhost"
	}
	r := model.DBProbeResult{
		Name: pg.Name, Backend: "postgres", Host: host, Port: port, Required: pg.Required,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		r.Error = "dial: " + err.Error()
		return r
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	if _, werr := conn.Write(pgSSLRequest); werr != nil {
		r.Error = "write SSLRequest: " + werr.Error()
		return r
	}
	body, rerr := readWith(conn, 1, timeout)
	if rerr != nil && rerr != io.EOF {
		r.Error = "read: " + rerr.Error()
		return r
	}
	if len(body) == 0 {
		r.Error = "server closed connection without replying"
		return r
	}
	if body[0] != 'S' && body[0] != 'N' {
		r.Error = fmt.Sprintf("unexpected byte 0x%02x; not a Postgres server", body[0])
		return r
	}
	r.Reachable = true
	r.HandshakeOK = true
	return r
}

// probeMySQL dials the server and reads its Initial Handshake packet
// (header + protocol-version byte + null-terminated server-version string).
// We never send a HandshakeResponse, so no auth.
func probeMySQL(my config.MySQLService, defaultTimeout time.Duration) model.DBProbeResult {
	timeout := resolveTimeout(my.TimeoutMS, defaultTimeout)
	host, port := my.Host, my.Port
	if host == "" {
		host = "localhost"
	}
	r := model.DBProbeResult{
		Name: my.Name, Backend: "mysql", Host: host, Port: port, Required: my.Required,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		r.Error = "dial: " + err.Error()
		return r
	}
	defer conn.Close()

	// Header: 3 bytes (payload length + sequence id 0).
	hdr, err := readWith(conn, 4, timeout)
	if err != nil || len(hdr) < 4 {
		r.Error = "read header: " + errString(err)
		return r
	}
	payloadLen := int(uint32(hdr[0]) | uint32(hdr[1])<<8 | uint32(hdr[2])<<16)
	if payloadLen == 0 || payloadLen > maxDBRead {
		r.Error = fmt.Sprintf("implausible handshake payload length %d", payloadLen)
		return r
	}
	body, err := readWith(conn, payloadLen, timeout)
	if err != nil && len(body) < 1 {
		r.Error = "read handshake body: " + errString(err)
		return r
	}
	if len(body) < 1 {
		r.Error = "empty handshake body"
		return r
	}
	if body[0] != mysqlHandshakeProtocolVersion {
		r.Error = fmt.Sprintf("expected MySQL protocol version 0x0a, got 0x%02x", body[0])
		return r
	}
	r.Reachable = true
	r.HandshakeOK = true
	// MySQL v10 handshake packet layout (after the 3-byte payload length and
	// 1-byte sequence id): protocol_version (1 byte), then a null-terminated
	// server version string. There are no reserved bytes between them.
	// References:
	//   https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_connection_phase.html
	//   https://mariadb.com/kb/en/client-server-protocol/
	if rest := body[1:]; len(rest) > 0 {
		if i := bytes.IndexByte(rest, 0); i >= 0 {
			r.ServerVersion = string(rest[:i])
		} else if len(rest) < 64 {
			r.ServerVersion = string(rest)
		} else {
			r.ServerVersion = string(rest[:64])
		}
	}
	return r
}

// probeRedis dials the server and sends a PING. Redis responds with +PONG\r\n
// (or -NOAUTH if a password is required — we treat that as HandshakeOK =
// false but Reachable = true, since the server is up but gated).
func probeRedis(rd config.RedisService, defaultTimeout time.Duration) model.DBProbeResult {
	timeout := resolveTimeout(rd.TimeoutMS, defaultTimeout)
	host, port := rd.Host, rd.Port
	if host == "" {
		host = "localhost"
	}
	r := model.DBProbeResult{
		Name: rd.Name, Backend: "redis", Host: host, Port: port, Required: rd.Required,
	}
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		r.Error = "dial: " + err.Error()
		return r
	}
	defer conn.Close()
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	if _, werr := conn.Write([]byte("*1\r\n$4\r\nPING\r\n")); werr != nil {
		r.Error = "write PING: " + werr.Error()
		return r
	}
	body, rerr := readWith(conn, maxDBRead, timeout)
	if rerr != nil && len(body) == 0 {
		r.Error = "read: " + rerr.Error()
		return r
	}
	line := string(body)
	r.Reachable = true
	switch {
	case bytes.HasPrefix(body, []byte("+PONG")):
		r.HandshakeOK = true
	case bytes.HasPrefix(body, []byte("-NOAUTH")) || bytes.HasPrefix(body, []byte("-ERR")):
		// Server is up but requires auth (or rejects our PING for some other
		// reason). The probe result is unambiguous: Redis is reachable.
		r.HandshakeOK = false
	default:
		r.HandshakeOK = false
		r.Error = "unexpected reply: " + trimResponse(line)
	}
	return r
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func trimResponse(s string) string {
	if len(s) > 64 {
		return s[:64] + "..."
	}
	return s
}

// ensure context is referenced (it would be if we ever wrapped the conn in
// a Dialer.Context-aware variant); keeps the import honest.
var _ = context.Background
