package share

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// ServerOptions configures a baseline-sharing server.
type ServerOptions struct {
	Host         string        // bind host; default 0.0.0.0
	Port         int           // bind port; 0 = random free port
	TTL          time.Duration // lifetime before auto-shutdown
	MaxDownloads int           // shut down after this many successful downloads
	UnsafeHTTP   bool          // serve plain HTTP (insecure) instead of TLS
	// PairingCode, when non-empty, requires the client to supply it as the
	// ?code= query parameter. Never logged, never in TXT records.
	PairingCode string
}

// Server is a running baseline-sharing server.
type Server struct {
	URL          string // full URL teammates should use (includes token)
	Fingerprint  string // TLS cert fingerprint (empty for unsafe HTTP)
	Pin          string // "sha256:..." pin string (empty for unsafe HTTP)
	TTL          time.Duration
	MaxDownloads int

	httpSrv   *http.Server
	listener  net.Listener
	downloads int64
	done      chan struct{}
	closeOnce sync.Once
}

// maxBaselineBytes caps the served payload size defensively.
const maxBaselineBytes = 1 << 20 // 1 MiB

// StartServer begins serving the given baseline bytes over a temporary server.
// It binds a listener, installs a single token-guarded route, and starts a
// goroutine that shuts the server down on TTL expiry. The caller should call
// Wait to block until shutdown and Close to stop early.
func StartServer(baseline []byte, opts ServerOptions) (*Server, error) {
	if len(baseline) == 0 {
		return nil, errors.New("baseline is empty")
	}
	if len(baseline) > maxBaselineBytes {
		return nil, fmt.Errorf("baseline too large (%d bytes, max %d)", len(baseline), maxBaselineBytes)
	}
	if opts.Host == "" {
		opts.Host = "0.0.0.0"
	}
	if opts.TTL <= 0 {
		opts.TTL = 10 * time.Minute
	}
	if opts.MaxDownloads <= 0 {
		opts.MaxDownloads = 20
	}

	token, err := randomToken(24)
	if err != nil {
		return nil, err
	}

	addr := net.JoinHostPort(opts.Host, fmt.Sprintf("%d", opts.Port))
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("bind %s: %w", addr, err)
	}

	srv := &Server{
		TTL:          opts.TTL,
		MaxDownloads: opts.MaxDownloads,
		listener:     ln,
		done:         make(chan struct{}),
	}

	scheme := "https"
	var tlsCfg *tls.Config
	if opts.UnsafeHTTP {
		scheme = "http"
	} else {
		cert, fp, certErr := ephemeralCert(opts.TTL)
		if certErr != nil {
			_ = ln.Close()
			return nil, certErr
		}
		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
			MinVersion:   tls.VersionTLS12,
		}
		srv.Fingerprint = fp
		srv.Pin = PinString(fp)
	}

	path := "/baseline/" + token
	mux := http.NewServeMux()

	// /resolve — returns {"url":"<full download URL>"} after code verification.
	// Used by LAN compare to discover the token URL without manual --pin paste.
	mux.HandleFunc("/resolve", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.NotFound(w, r)
			return
		}
		if r.ContentLength > 0 {
			http.Error(w, "uploads not allowed", http.StatusMethodNotAllowed)
			return
		}
		if opts.PairingCode != "" {
			supplied := r.URL.Query().Get("code")
			if !constantTimeEqual(supplied, opts.PairingCode) {
				http.Error(w, "invalid pairing code", http.StatusForbidden)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"url":%q}`, srv.URL)
	})

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Only the exact token path with GET is served; everything else 404s.
		if r.Method != http.MethodGet || r.URL.Path != path {
			http.NotFound(w, r)
			return
		}
		// Enforce single-purpose: reject bodies/uploads.
		if r.ContentLength > 0 {
			http.Error(w, "uploads not allowed", http.StatusMethodNotAllowed)
			return
		}
		// Pairing-code gate: if configured, the ?code= param must match.
		// The code is compared using a constant-time approach to avoid timing
		// side-channels. Do NOT log the supplied code.
		if opts.PairingCode != "" {
			supplied := r.URL.Query().Get("code")
			if !constantTimeEqual(supplied, opts.PairingCode) {
				http.Error(w, "invalid pairing code", http.StatusForbidden)
				return
			}
		}
		n := atomic.AddInt64(&srv.downloads, 1)
		if n > int64(srv.MaxDownloads) {
			http.Error(w, "download limit reached", http.StatusGone)
			go srv.Close()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(baseline)
		if n == int64(srv.MaxDownloads) {
			go srv.Close()
		}
	})

	srv.httpSrv = &http.Server{
		Handler:           mux,
		TLSConfig:         tlsCfg,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       15 * time.Second,
		MaxHeaderBytes:    1 << 16,
	}

	port := ln.Addr().(*net.TCPAddr).Port
	host := opts.Host
	if host == "0.0.0.0" || host == "::" {
		host = localIP()
	}
	srv.URL = fmt.Sprintf("%s://%s:%d%s", scheme, host, port, path)

	go func() {
		var serveErr error
		if opts.UnsafeHTTP {
			serveErr = srv.httpSrv.Serve(ln)
		} else {
			serveErr = srv.httpSrv.ServeTLS(ln, "", "")
		}
		_ = serveErr // http.ErrServerClosed on normal shutdown
	}()

	// Auto-shutdown on TTL.
	go func() {
		select {
		case <-time.After(opts.TTL):
			srv.Close()
		case <-srv.done:
		}
	}()

	return srv, nil
}

// Downloads returns the number of download attempts served so far.
func (s *Server) Downloads() int { return int(atomic.LoadInt64(&s.downloads)) }

// Wait blocks until the server has shut down.
func (s *Server) Wait() { <-s.done }

// Close shuts the server down and releases resources. Safe to call multiple
// times and from multiple goroutines.
func (s *Server) Close() {
	s.closeOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.httpSrv.Shutdown(ctx)
		_ = s.listener.Close()
		close(s.done)
	})
}

// localIP returns a best-effort non-loopback IPv4 address for display, or
// "127.0.0.1" if none is found.
func localIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4.String()
			}
		}
	}
	return "127.0.0.1"
}

// constantTimeEqual compares two strings in constant time to prevent
// timing attacks on pairing codes.
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// PairingCode generates a random 6-digit decimal code using crypto/rand.
// The code is printed to the operator only; never logged, never in mDNS TXT.
func PairingCode() (string, error) {
	n, err := cryptoRandInt(1_000_000)
	if err != nil {
		return "", fmt.Errorf("generate pairing code: %w", err)
	}
	return fmt.Sprintf("%06d", n), nil
}

// LocalIP is the exported version for use in the LAN advertiser.
func LocalIP() string { return localIP() }

// cryptoRandInt returns a random int64 in [0, max).
func cryptoRandInt(max int64) (int64, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(max))
	if err != nil {
		return 0, err
	}
	return n.Int64(), nil
}
