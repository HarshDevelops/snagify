// Package lan implements mDNS-based LAN discovery for Snagify shares.
// It advertises a share on _snagify._tcp and discovers peers on the same
// LAN/VPN without any cloud or relay server.
package lan

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grandcat/zeroconf"
)

const (
	// ServiceType is the mDNS service type used for Snagify shares.
	ServiceType = "_snagify._tcp"
	// Domain is the mDNS domain for local discovery.
	Domain = "local."
)

// ShareMeta is the metadata advertised in the mDNS TXT record.
// Never include secrets, env keys, baseline content, or pairing codes.
type ShareMeta struct {
	Repo        string
	Project     string
	Version     string
	Mode        string // "open" or "code"
	Fingerprint string // TLS cert fingerprint so client can pin without --pin flag
	ExpiresAt   int64  // Unix timestamp
	Port        int    // the HTTPS port where baseline is served
}

// Advertiser publishes a Snagify share via mDNS.
type Advertiser struct {
	server       *zeroconf.Server
	instanceName string
}

// Advertise starts an mDNS advertisement for a LAN share. Cancel the returned
// context to stop advertising.
func Advertise(meta ShareMeta) (*Advertiser, error) {
	txtRecords := buildTXT(meta)
	instance := sanitiseInstance(meta.Repo)

	srv, err := zeroconf.Register(
		instance,
		ServiceType,
		Domain,
		meta.Port,
		txtRecords,
		nil, // all interfaces
	)
	if err != nil {
		return nil, fmt.Errorf("mDNS advertise: %w", err)
	}
	return &Advertiser{server: srv, instanceName: instance}, nil
}

// Stop stops advertising. Safe to call multiple times.
func (a *Advertiser) Stop() {
	if a.server != nil {
		a.server.Shutdown()
	}
}

// buildTXT returns the mDNS TXT key=value pairs for a share.
// NEVER include secrets, env key lists, baseline content, or pairing codes.
func buildTXT(meta ShareMeta) []string {
	return []string{
		"repo=" + meta.Repo,
		"project=" + meta.Project,
		"version=" + meta.Version,
		"mode=" + meta.Mode,
		"fingerprint=" + meta.Fingerprint,
		"expires_at=" + strconv.FormatInt(meta.ExpiresAt, 10),
	}
}

// sanitiseInstance ensures the mDNS instance name is safe (no dots).
func sanitiseInstance(name string) string {
	s := strings.ReplaceAll(name, ".", "-")
	if s == "" {
		return "snagify"
	}
	return "snagify-" + s
}

// DiscoveredShare is a share discovered via mDNS.
type DiscoveredShare struct {
	Repo        string
	Project     string
	Version     string
	Mode        string // "open" or "code"
	Fingerprint string
	ExpiresAt   int64
	Host        string
	Port        int
}

// URL returns the HTTPS URL of the baseline route.
// The token is not in the discovery metadata; the share server provides a
// separate endpoint for code-authenticated download.
func (d DiscoveredShare) BaseURL() string {
	return fmt.Sprintf("https://%s:%d", d.Host, d.Port)
}

// IsExpired reports whether the share has passed its expiry timestamp.
func (d DiscoveredShare) IsExpired() bool {
	return d.ExpiresAt > 0 && time.Now().Unix() > d.ExpiresAt
}

// Discover browses for active Snagify LAN shares, returning those that match
// repoFilter (or all shares when repoFilter is empty). It waits up to timeout
// for responses.
func Discover(ctx context.Context, repoFilter string, timeout time.Duration) ([]DiscoveredShare, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("mDNS resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	var shares []DiscoveredShare

	go func() {
		for entry := range entries {
			meta := parseTXT(entry.Text)
			if repoFilter != "" && meta.Repo != repoFilter {
				continue
			}
			host := entry.HostName
			// Prefer IPv4.
			if len(entry.AddrIPv4) > 0 {
				host = entry.AddrIPv4[0].String()
			} else if len(entry.AddrIPv6) > 0 {
				host = entry.AddrIPv6[0].String()
			}
			shares = append(shares, DiscoveredShare{
				Repo:        meta.Repo,
				Project:     meta.Project,
				Version:     meta.Version,
				Mode:        meta.Mode,
				Fingerprint: meta.Fingerprint,
				ExpiresAt:   meta.ExpiresAt,
				Host:        host,
				Port:        entry.Port,
			})
		}
	}()

	browseCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := resolver.Browse(browseCtx, ServiceType, Domain, entries); err != nil {
		return nil, fmt.Errorf("mDNS browse: %w", err)
	}

	<-browseCtx.Done()
	return shares, nil
}

// parseTXT extracts ShareMeta from mDNS TXT key=value pairs.
func parseTXT(records []string) ShareMeta {
	m := make(map[string]string, len(records))
	for _, r := range records {
		if i := strings.IndexByte(r, '='); i > 0 {
			m[r[:i]] = r[i+1:]
		}
	}
	exp, _ := strconv.ParseInt(m["expires_at"], 10, 64)
	return ShareMeta{
		Repo:        m["repo"],
		Project:     m["project"],
		Version:     m["version"],
		Mode:        m["mode"],
		Fingerprint: m["fingerprint"],
		ExpiresAt:   exp,
	}
}
