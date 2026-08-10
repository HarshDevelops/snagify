package probe

import (
	"net"
	"strings"
	"testing"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
)

// postgresFakeServer answers SSLRequest with "N" and waits for the client to
// disconnect — it mirrors real Postgres' behavior of declining a cleartext
// SSL upgrade while proving the wire protocol is in play.
func postgresFakeServer(t *testing.T) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetReadDeadline(time.Now().Add(time.Second))
				buf := make([]byte, 8)
				_, _ = c.Read(buf)
				_, _ = c.Write([]byte("N"))
			}(c)
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func mysqlFakeServerBanner(t *testing.T, banner []byte) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = c.Write(banner)
			}(c)
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func redisFakeServer(t *testing.T, reply string) (addr string, cleanup func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_ = c.SetReadDeadline(time.Now().Add(time.Second))
				// Read PING and discard.
				scratch := make([]byte, 64)
				_, _ = c.Read(scratch)
				_, _ = c.Write([]byte(reply))
			}(c)
		}
	}()
	return ln.Addr().String(), func() {
		_ = ln.Close()
		<-done
	}
}

func TestProbePostgres_ServerAcceptsOrDeclinesSSL(t *testing.T) {
	addr, stop := postgresFakeServer(t)
	defer stop()
	host, port := splitHostPort(t, addr)

	got := probePostgres(config.PostgresService{
		Name: "pg", Host: host, Port: port, Required: true,
	}, time.Second)

	if !got.Reachable || !got.HandshakeOK {
		t.Fatalf("expected reachable+handshake ok, got %+v (err=%s)", got, got.Error)
	}
	if got.Backend != "postgres" {
		t.Errorf("backend=%q, want postgres", got.Backend)
	}
}

func TestProbePostgres_ConnectionRefused(t *testing.T) {
	got := probePostgres(config.PostgresService{
		Name: "pg", Host: "127.0.0.1", Port: 1, Required: true,
	}, 200*time.Millisecond)
	if got.Reachable {
		t.Errorf("expected unreachable on closed port, got %+v", got)
	}
	if got.Error == "" {
		t.Errorf("expected error message, got empty")
	}
}

func TestProbeMySQL_HandshakeRead(t *testing.T) {
	// MySQL handshake: 3-byte header (payload_len=51, seq=0) + protocol = 0x0a
	// + 2 bytes reserved + null-terminated server version + connection id...
	// Just enough to satisfy the parser: len=4, payload = 0x0a + "5.7\0".
	banner := []byte{
		0x05, 0x00, 0x00, // payload length = 5
		0x00,                // sequence id = 0
		0x0a,                // protocol version 10
		'5', '.', '7', '\x00',
	}
	addr, stop := mysqlFakeServerBanner(t, banner)
	defer stop()
	host, port := splitHostPort(t, addr)

	got := probeMySQL(config.MySQLService{Name: "my", Host: host, Port: port, Required: true}, time.Second)

	if !got.Reachable || !got.HandshakeOK {
		t.Fatalf("expected reachable+handshake ok, got %+v (err=%s)", got, got.Error)
	}
	if got.ServerVersion != "5.7" {
		t.Errorf("server version = %q, want 5.7", got.ServerVersion)
	}
}

func TestProbeMySQL_WrongProtocolVersion(t *testing.T) {
	banner := []byte{
		0x02, 0x00, 0x00,
		0x00,
		0xff, 'x',
	}
	addr, stop := mysqlFakeServerBanner(t, banner)
	defer stop()
	host, port := splitHostPort(t, addr)

	got := probeMySQL(config.MySQLService{Name: "my", Host: host, Port: port, Required: false}, time.Second)

	if got.Reachable {
		t.Errorf("expected handshake to fail, got %+v", got)
	}
	if !strings.Contains(got.Error, "0x") {
		t.Errorf("expected a hex error message, got %q", got.Error)
	}
}

func TestProbeRedis_PONG(t *testing.T) {
	addr, stop := redisFakeServer(t, "+PONG\r\n")
	defer stop()
	host, port := splitHostPort(t, addr)

	got := probeRedis(config.RedisService{Name: "redis", Host: host, Port: port, Required: true}, time.Second)

	if !got.Reachable {
		t.Fatalf("expected reachable, got %+v", got)
	}
	if !got.HandshakeOK {
		t.Fatalf("expected handshake ok (PONG), got %+v", got)
	}
}

func TestProbeRedis_NOAUTHStillReachable(t *testing.T) {
	addr, stop := redisFakeServer(t, "-NOAUTH Authentication required.\r\n")
	defer stop()
	host, port := splitHostPort(t, addr)

	got := probeRedis(config.RedisService{Name: "redis", Host: host, Port: port, Required: false}, time.Second)

	if !got.Reachable {
		t.Errorf("expected reachable even when NOAUTH returned; got %+v", got)
	}
	if got.HandshakeOK {
		t.Errorf("expected handshake-ok=false on NOAUTH, got %+v", got)
	}
	if got.Error != "" {
		t.Errorf("expected empty error on NOAUTH (it's a known response shape), got %q", got.Error)
	}
}

func splitHostPort(t *testing.T, addr string) (string, int) {
	t.Helper()
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		t.Fatalf("split %q: %v", addr, err)
	}
	port := 0
	for _, c := range portStr {
		if c < '0' || c > '9' {
			t.Fatalf("non-numeric port %q", portStr)
		}
		port = port*10 + int(c-'0')
	}
	return host, port
}
