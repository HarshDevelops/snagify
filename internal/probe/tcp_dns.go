package probe

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

// lookupEnv is indirected for testing.
var lookupEnv = os.LookupEnv

func probeTCP(svc config.TCPService, defaultTimeout time.Duration) model.TCPProbeResult {
	timeout := defaultTimeout
	if svc.TimeoutMS > 0 {
		timeout = time.Duration(svc.TimeoutMS) * time.Millisecond
	}
	host := svc.Host
	if host == "" {
		host = "localhost"
	}

	r := model.TCPProbeResult{
		Name:     svc.Name,
		Host:     host,
		Port:     svc.Port,
		Required: svc.Required,
	}

	addr := net.JoinHostPort(host, fmt.Sprintf("%d", svc.Port))
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		r.Reachable = false
		r.Error = err.Error()
		return r
	}
	_ = conn.Close()
	r.Reachable = true
	return r
}

func probeDNS(resolver Resolver, host string, timeout time.Duration) model.DNSProbeResult {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	r := model.DNSProbeResult{Host: host}
	addrs, err := resolver.LookupHost(ctx, host)
	if err != nil || len(addrs) == 0 {
		r.Resolved = false
		if err != nil {
			r.Error = err.Error()
		} else {
			r.Error = "no addresses returned"
		}
		return r
	}
	r.Resolved = true
	return r
}
