package probe

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

// probeTLS performs a TLS handshake and inspects the leaf certificate. It does
// not disable verification unless insecure is set (the CLI prints a loud
// warning in that case).
func probeTLS(ep config.TLSEndpoint, timeout time.Duration, insecure bool) model.TLSProbeResult {
	port := ep.Port
	if port == 0 {
		port = 443
	}
	serverName := ep.ServerName
	if serverName == "" {
		serverName = ep.Host
	}

	r := model.TLSProbeResult{
		Name:       ep.Name,
		Host:       ep.Host,
		Port:       port,
		ServerName: serverName,
		Required:   ep.Required,
	}

	dialer := &net.Dialer{Timeout: timeout}
	addr := net.JoinHostPort(ep.Host, fmt.Sprintf("%d", port))

	// First, attempt a verified handshake.
	conn, err := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
		ServerName: serverName,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		// Retry without verification to distinguish hostname/expiry failures
		// from total handshake failure, but only inspect — never trust.
		insecureConn, ierr := tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{
			ServerName:         serverName,
			InsecureSkipVerify: true, //nolint:gosec // inspection only; not trusted
			MinVersion:         tls.VersionTLS12,
		})
		if ierr != nil {
			r.HandshakeOK = false
			r.Error = err.Error()
			return r
		}
		// Handshake works but verification failed (hostname/expiry/issuer).
		r.HandshakeOK = true
		r.HostnameOK = false
		inspectCert(&r, insecureConn)
		_ = insecureConn.Close()
		if r.Error == "" {
			r.Error = err.Error()
		}
		if insecure {
			// Caller opted into insecure inspection; keep results but flag.
			r.HostnameOK = false
		}
		return r
	}
	defer conn.Close()

	r.HandshakeOK = true
	r.HostnameOK = true
	inspectCert(&r, conn)
	return r
}

// inspectCert records issuer and expiry from the peer's leaf certificate.
func inspectCert(r *model.TLSProbeResult, conn *tls.Conn) {
	certs := conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return
	}
	leaf := certs[0]
	r.Issuer = leaf.Issuer.CommonName
	r.NotAfter = leaf.NotAfter.UTC().Format(time.RFC3339)
	days := int(time.Until(leaf.NotAfter).Hours() / 24)
	r.DaysToExpiry = days
	if time.Now().After(leaf.NotAfter) {
		r.Expired = true
	}
}
