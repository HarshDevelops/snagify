package share

import (
	"crypto/tls"
	"net/http"
	"strings"
	"testing"
	"time"
)

func startTestServer(t *testing.T, payload string) *Server {
	t.Helper()
	srv, err := StartServer([]byte(payload), ServerOptions{
		Host:         "127.0.0.1",
		TTL:          5 * time.Second,
		MaxDownloads: 5,
	})
	if err != nil {
		t.Fatalf("StartServer: %v", err)
	}
	t.Cleanup(srv.Close)
	return srv
}

func TestFetch_CorrectPinWorks(t *testing.T) {
	srv := startTestServer(t, `{"hello":"world"}`)

	data, err := Fetch(FetchOptions{URL: srv.URL, Pin: srv.Pin, Timeout: 3 * time.Second})
	if err != nil {
		t.Fatalf("Fetch with correct pin failed: %v", err)
	}
	if string(data) != `{"hello":"world"}` {
		t.Errorf("unexpected payload: %s", data)
	}
}

func TestFetch_WrongPinFails(t *testing.T) {
	srv := startTestServer(t, `{"a":1}`)

	bogus := "sha256:" + strings.Repeat("AB:", 31) + "AB"
	_, err := Fetch(FetchOptions{URL: srv.URL, Pin: bogus, Timeout: 3 * time.Second})
	if err == nil {
		t.Fatal("expected failure with wrong pin")
	}
}

func TestFetch_WrongTokenFails(t *testing.T) {
	srv := startTestServer(t, `{"a":1}`)

	// Replace the token segment with a bogus one.
	idx := strings.LastIndex(srv.URL, "/")
	wrongURL := srv.URL[:idx+1] + "deadbeefdeadbeef"

	_, err := Fetch(FetchOptions{URL: wrongURL, Pin: srv.Pin, Timeout: 3 * time.Second})
	if err == nil {
		t.Fatal("expected failure with wrong token")
	}
}

func TestServer_NoUploadRoute(t *testing.T) {
	srv := startTestServer(t, `{"a":1}`)

	client := &http.Client{
		Timeout: 3 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //nolint:gosec // test only
		},
	}

	// POST to the valid path must not be accepted.
	resp, err := client.Post(srv.URL, "application/json", strings.NewReader(`{"x":1}`))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Errorf("upload via POST unexpectedly accepted (status %d)", resp.StatusCode)
	}

	// An arbitrary unknown route must 404.
	resp2, err := client.Get(strings.Replace(srv.URL, "/baseline/", "/secret/", 1))
	if err != nil {
		t.Fatalf("get unknown: %v", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want 404", resp2.StatusCode)
	}
}

func TestFetch_PlainHTTPRejectedWithoutFlag(t *testing.T) {
	_, err := Fetch(FetchOptions{URL: "http://127.0.0.1:1/baseline/x", Pin: "", Timeout: time.Second})
	if err == nil {
		t.Fatal("expected refusal of plain HTTP without --unsafe-http")
	}
}

func TestFingerprintFormat(t *testing.T) {
	fp := Fingerprint([]byte("test"))
	if len(strings.Split(fp, ":")) != 32 {
		t.Errorf("expected 32 hex octets, got %q", fp)
	}
	if NormalizePin("sha256:"+fp) != fp {
		t.Error("NormalizePin should strip sha256: prefix and match")
	}
}
