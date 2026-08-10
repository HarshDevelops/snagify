// Package probe performs active connectivity probes (TCP, DNS, HTTP, TLS,
// and unauthenticated database wire-protocol probes) and captures proxy env
// presence. All probes are read-only: no credentials are sent, HTTP response
// bodies and database payload contents are never read or stored, and the
// only writes to a database socket are the protocol pings required to elicit
// an authentication handshake. No password, key, or query is transmitted.
package probe

import (
	"context"
	"net"
	"time"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

// Resolver abstracts DNS lookups so tests can inject a fake.
type Resolver interface {
	LookupHost(ctx context.Context, host string) ([]string, error)
}

// Options configures a probe run.
type Options struct {
	DefaultTimeout time.Duration
	Resolver       Resolver
	SkipNetwork    bool // skip DNS + HTTP
	SkipTLS        bool // skip TLS handshakes
	InsecureProbe  bool // allow TLS verification skip (with loud warning at CLI)
}

// proxyVars are the environment variables whose presence (not value) is
// reported.
var proxyVars = []string{
	"HTTP_PROXY", "HTTPS_PROXY", "NO_PROXY",
	"http_proxy", "https_proxy", "no_proxy",
}

// Run executes all probes declared in cfg and returns the results.
func Run(cfg config.Config, opts Options) model.ProbeResults {
	if opts.DefaultTimeout <= 0 {
		opts.DefaultTimeout = 2 * time.Second
	}
	if opts.Resolver == nil {
		opts.Resolver = net.DefaultResolver
	}

	res := model.ProbeResults{Ran: true}

	for _, svc := range cfg.Services.TCP {
		res.TCP = append(res.TCP, probeTCP(svc, opts.DefaultTimeout))
	}

	if !opts.SkipNetwork {
		for _, host := range cfg.Network.DNS {
			res.DNS = append(res.DNS, probeDNS(opts.Resolver, host, opts.DefaultTimeout))
		}
		for _, h := range cfg.Network.HTTP {
			res.HTTP = append(res.HTTP, probeHTTP(h, opts.DefaultTimeout))
		}
	}

	if !opts.SkipTLS {
		for _, ep := range cfg.TLS.Endpoints {
			res.TLS = append(res.TLS, probeTLS(ep, opts.DefaultTimeout, opts.InsecureProbe))
		}
	}

	if len(cfg.Databases.Postgres) > 0 || len(cfg.Databases.MySQL) > 0 || len(cfg.Databases.Redis) > 0 {
		res.DB = runDBProbes(cfg, opts.DefaultTimeout)
	}

	if cfg.Network.Proxy.CaptureEnv {
		res.Proxy = captureProxy()
	}

	return res
}

func captureProxy() model.ProxyInfo {
	vars := map[string]bool{}
	for _, v := range proxyVars {
		_, set := lookupEnv(v)
		vars[v] = set
	}
	return model.ProxyInfo{Captured: true, Vars: vars}
}
