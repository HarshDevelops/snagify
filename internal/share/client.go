package share

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// ErrPinMismatch is returned when the server certificate fingerprint does not
// match the expected pin.
var ErrPinMismatch = errors.New("certificate fingerprint does not match --pin")

// FetchOptions configures a baseline fetch.
type FetchOptions struct {
	URL         string
	Pin         string // expected fingerprint, with or without "sha256:" prefix
	UnsafeHTTP  bool   // allow plain HTTP (no pinning); prints caller's warning
	Timeout     time.Duration
	PairingCode string // if non-empty, appended as ?code=XXX query parameter
}

// Fetch downloads a baseline from a sharing server. For HTTPS it pins the
// server certificate fingerprint exactly and fails loudly on mismatch. The
// response body size is capped.
func Fetch(opts FetchOptions) ([]byte, error) {
	rawURL := strings.TrimSpace(opts.URL)
	// Append pairing code as query parameter if provided.
	if opts.PairingCode != "" {
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		rawURL = rawURL + sep + "code=" + opts.PairingCode
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid --from URL: %w", err)
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 15 * time.Second
	}

	switch u.Scheme {
	case "https":
		if opts.Pin == "" {
			return nil, errors.New("--pin is required for https sharing")
		}
		return fetchTLS(u.String(), opts.Pin, opts.Timeout)
	case "http":
		if !opts.UnsafeHTTP {
			return nil, errors.New("refusing plain HTTP without --unsafe-http")
		}
		return fetchPlain(u.String(), opts.Timeout)
	default:
		return nil, fmt.Errorf("unsupported scheme %q (use https)", u.Scheme)
	}
}

func fetchTLS(rawurl, pin string, timeout time.Duration) ([]byte, error) {
	want := NormalizePin(pin)

	verify := func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(rawCerts) == 0 {
			return ErrPinMismatch
		}
		got := Fingerprint(rawCerts[0])
		if !equalPins(got, want) {
			return fmt.Errorf("%w (server: sha256:%s)", ErrPinMismatch, got)
		}
		return nil
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			// We do our own verification via VerifyPeerCertificate, pinning the
			// exact fingerprint. The cert is self-signed by design.
			InsecureSkipVerify:    true, //nolint:gosec // pinned below
			VerifyPeerCertificate: verify,
			MinVersion:            tls.VersionTLS12,
		},
		DialContext: (&net.Dialer{Timeout: timeout}).DialContext,
	}
	return doGet(tr, rawurl, timeout)
}

func fetchPlain(rawurl string, timeout time.Duration) ([]byte, error) {
	tr := &http.Transport{
		DialContext: (&net.Dialer{Timeout: timeout}).DialContext,
	}
	return doGet(tr, rawurl, timeout)
}

func doGet(tr *http.Transport, rawurl string, timeout time.Duration) ([]byte, error) {
	client := &http.Client{Transport: tr, Timeout: timeout}
	resp, err := client.Get(rawurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned %s", resp.Status)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBaselineBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxBaselineBytes {
		return nil, errors.New("baseline response exceeds size limit")
	}
	return data, nil
}
