package share

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestPairingCode_Format(t *testing.T) {
	for i := 0; i < 10; i++ {
		code, err := PairingCode()
		if err != nil {
			t.Fatalf("PairingCode: %v", err)
		}
		if len(code) != 6 {
			t.Errorf("expected 6-digit code, got %q (len %d)", code, len(code))
		}
		for _, c := range code {
			if c < '0' || c > '9' {
				t.Errorf("code contains non-digit %q", c)
			}
		}
	}
}

func startCodeServer(t *testing.T, code string) *Server {
	t.Helper()
	srv, err := StartServer([]byte(`{"x":1}`), ServerOptions{
		Host:         "127.0.0.1",
		TTL:          5 * time.Second,
		MaxDownloads: 10,
		PairingCode:  code,
	})
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	t.Cleanup(srv.Close)
	return srv
}

func TestPairingCode_RequiredModeRejectsWrongCode(t *testing.T) {
	srv := startCodeServer(t, "123456")

	// Wrong code should get 403.
	_, err := Fetch(FetchOptions{URL: srv.URL, Pin: srv.Pin, PairingCode: "999999", Timeout: 3 * time.Second})
	if err == nil {
		t.Fatal("expected failure with wrong pairing code")
	}
}

func TestPairingCode_RequiredModeMissingCode(t *testing.T) {
	srv := startCodeServer(t, "123456")

	// No code should get 403.
	_, err := Fetch(FetchOptions{URL: srv.URL, Pin: srv.Pin, Timeout: 3 * time.Second})
	if err == nil {
		t.Fatal("expected failure with missing pairing code")
	}
}

func TestPairingCode_CorrectCodeAllowsDownload(t *testing.T) {
	srv := startCodeServer(t, "123456")

	data, err := Fetch(FetchOptions{URL: srv.URL, Pin: srv.Pin, PairingCode: "123456", Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Fetch with correct code failed: %v", err)
	}
	if string(data) != `{"x":1}` {
		t.Errorf("unexpected payload: %s", data)
	}
}

func TestPairingCode_OpenModeAllowsWithoutCode(t *testing.T) {
	srv, err := StartServer([]byte(`{"open":true}`), ServerOptions{
		Host:         "127.0.0.1",
		TTL:          5 * time.Second,
		MaxDownloads: 10,
		PairingCode:  "", // open mode
	})
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	defer srv.Close()

	data, err := Fetch(FetchOptions{URL: srv.URL, Pin: srv.Pin, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("open mode fetch failed: %v", err)
	}
	if string(data) != `{"open":true}` {
		t.Errorf("unexpected payload: %s", data)
	}
}

func TestResolveEndpoint_WithCode(t *testing.T) {
	srv := startCodeServer(t, "654321")

	// Build host:port.
	host := strings.TrimPrefix(strings.TrimPrefix(srv.URL, "https://"), "http://")
	if i := strings.LastIndex(host, "/"); i >= 0 {
		host = host[:i]
	}
	if i := strings.Index(host, "/"); i >= 0 {
		host = host[:i]
	}

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
		},
	}

	baseURL := fmt.Sprintf("https://%s", host)
	// Wrong code → 403.
	resp, err := client.Get(baseURL + "/resolve?code=000000")
	if err != nil {
		t.Fatalf("resolve GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("expected 403 for wrong code, got %d", resp.StatusCode)
	}

	// Correct code → 200 with URL.
	resp2, err := client.Get(baseURL + "/resolve?code=654321")
	if err != nil {
		t.Fatalf("resolve GET: %v", err)
	}
	resp2.Body.Close()
	if resp2.StatusCode != http.StatusOK {
		t.Errorf("expected 200 for correct code, got %d", resp2.StatusCode)
	}
}
