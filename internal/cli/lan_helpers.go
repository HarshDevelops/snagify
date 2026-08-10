package cli

import (
	"net/url"
	"strconv"
	"time"

	"github.com/harshdevelops/snagify/internal/lan"
)

// version is injected from main at startup; set here as fallback.
var version = "0.5.0"

// lanAdvertiser wraps lan.Advertiser with a simple stop method.
type lanAdvertiser struct {
	adv *lan.Advertiser
}

func (a *lanAdvertiser) stop() {
	if a != nil && a.adv != nil {
		a.adv.Stop()
	}
}

// advertiseLAN starts an mDNS advertisement and returns an advertiser.
func advertiseLAN(repo, project, ver, mode, fingerprint string, port int, ttl time.Duration) (*lanAdvertiser, error) {
	expiresAt := time.Now().Add(ttl).Unix()
	adv, err := lan.Advertise(lan.ShareMeta{
		Repo:        repo,
		Project:     project,
		Version:     ver,
		Mode:        mode,
		Fingerprint: fingerprint,
		ExpiresAt:   expiresAt,
		Port:        port,
	})
	if err != nil {
		return nil, err
	}
	return &lanAdvertiser{adv: adv}, nil
}

// parseHostPort extracts the port from a URL string like https://host:port/path.
func parseHostPort(rawURL string) (host string, port int) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", 0
	}
	host = u.Hostname()
	p, _ := strconv.Atoi(u.Port())
	return host, p
}
