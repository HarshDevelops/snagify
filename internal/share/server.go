package share

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
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
