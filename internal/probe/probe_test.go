package probe

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
)

func TestProbeTCP_ReachableAndClosed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	port := ln.Addr().(*net.TCPAddr).Port

	open := probeTCP(config.TCPService{Name: "svc", Host: "127.0.0.1", Port: port, Required: true}, time.Second)
	if !open.Reachable {
		t.Errorf("expected reachable on open port, err=%s", open.Error)
	}

	// A port with nothing listening.
	ln2, _ := net.Listen("tcp", "127.0.0.1:0")
	closedPort := ln2.Addr().(*net.TCPAddr).Port
	ln2.Close()
	closed := probeTCP(config.TCPService{Name: "svc", Host: "127.0.0.1", Port: closedPort, Required: true}, 500*time.Millisecond)
	if closed.Reachable {
		t.Error("expected unreachable on closed port")
	}
}

type fakeResolver struct {
	hosts map[string][]string
}

func (f fakeResolver) LookupHost(_ context.Context, host string) ([]string, error) {
	if addrs, ok := f.hosts[host]; ok {
		return addrs, nil
	}
	return nil, errors.New("no such host")
}

func TestProbeDNS_SuccessAndFailure(t *testing.T) {
	r := fakeResolver{hosts: map[string][]string{"good.example": {"93.184.216.34"}}}

	ok := probeDNS(r, "good.example", time.Second)
	if !ok.Resolved {
		t.Error("expected resolved")
	}
	bad := probeDNS(r, "missing.example", time.Second)
	if bad.Resolved {
		t.Error("expected unresolved")
	}
}

func TestProbeHTTP_StatusMatchAndMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("SECRET_BODY_SHOULD_NOT_BE_READ"))
	}))
	defer srv.Close()

	ok := probeHTTP(config.HTTPProbe{Name: "x", URL: srv.URL, Method: "HEAD", ExpectedStatus: 200, Required: true}, 2*time.Second)
	if !ok.OK {
		t.Errorf("expected OK, got status=%d err=%s", ok.GotStatus, ok.Error)
	}

	mismatch := probeHTTP(config.HTTPProbe{Name: "x", URL: srv.URL, ExpectedStatus: 404, Required: true}, 2*time.Second)
	if mismatch.OK {
		t.Error("expected mismatch when expecting 404")
	}
}

func TestProbeTLS_ValidAndHostnameMismatch(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	host, portStr, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "https://"))
	port, _ := strconv.Atoi(portStr)

	// httptest TLS server cert is for "example.com" / 127.0.0.1; the default
	// httptest cert includes 127.0.0.1 and example.com SANs. Use 127.0.0.1.
	good := probeTLS(config.TLSEndpoint{Name: "x", Host: host, Port: port, ServerName: "example.com", Required: true}, 2*time.Second, false)
	if !good.HandshakeOK {
		t.Errorf("expected handshake ok, err=%s", good.Error)
	}

	// Wrong server name -> hostname verification fails but handshake succeeds.
	bad := probeTLS(config.TLSEndpoint{Name: "x", Host: host, Port: port, ServerName: "wrong.invalid", Required: true}, 2*time.Second, false)
	if bad.HostnameOK {
		t.Error("expected hostname verification to fail for wrong server name")
	}
}

func TestCaptureProxy_PresenceNotValue(t *testing.T) {
	const secret = "http://user:pass@proxy.internal:8080"
	t.Setenv("HTTPS_PROXY", secret)
	t.Setenv("HTTP_PROXY", "")

	info := captureProxy()
	if !info.Captured {
		t.Fatal("expected captured=true")
	}
	if !info.Vars["HTTPS_PROXY"] {
		t.Error("expected HTTPS_PROXY marked set")
	}
	// The value must never appear anywhere in the captured struct.
	for k, v := range info.Vars {
		if strings.Contains(k, secret) {
			t.Errorf("proxy value leaked in key %q", k)
		}
		_ = v
	}
}
