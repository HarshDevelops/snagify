package check

import (
	"fmt"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/report"
)

// ProbeReport builds a report from only the active probe results in snap,
// for the `snagify probe` command.
func ProbeReport(snap model.Snapshot, cfg config.Config) report.Report {
	name := cfg.Project.Name
	if name == "" {
		name = snap.Project.Name
	}
	r := report.Report{
		Project:    name,
		FoundLabel: "found",
		ExpLabel:   "expected",
	}
	checkProbes(&r, snap, cfg)
	if r.Failed() {
		r.Status = "Probes found likely blockers"
	} else {
		r.Status = "All probes passed"
	}
	return r
}

// checkProbes folds active probe results into the report. Required failures are
// critical; optional failures are warnings.
func checkProbes(r *report.Report, snap model.Snapshot, cfg config.Config) {
	p := snap.Probes
	if p == nil || !p.Ran {
		return
	}

	for _, t := range p.TCP {
		if t.Reachable {
			continue
		}
		sev := report.Warning
		var blocker string
		if t.Required {
			sev = report.Critical
			blocker = fmt.Sprintf("service %s (%s:%d) is unreachable", t.Name, t.Host, t.Port)
		}
		r.Add(report.Item{
			Category: "Service", Name: t.Name,
			Found: fmt.Sprintf("%s:%d unreachable", t.Host, t.Port), Expected: "reachable",
			Severity: sev, Blocker: blocker,
		})
	}

	for _, d := range p.DNS {
		if d.Resolved {
			continue
		}
		r.Add(report.Item{
			Category: "DNS", Name: d.Host,
			Found: "lookup failed", Expected: "resolves", Severity: report.Critical,
			Blocker: fmt.Sprintf("DNS lookup for %s failed", d.Host),
		})
	}

	for _, h := range p.HTTP {
		if h.OK {
			continue
		}
		sev := report.Warning
		if h.Required {
			sev = report.Critical
		}
		found := h.Error
		if found == "" {
			found = fmt.Sprintf("status %d", h.GotStatus)
		}
		item := report.Item{
			Category: "HTTP", Name: h.Name,
			Found: found, Expected: fmt.Sprintf("status %d", h.ExpectedStatus), Severity: sev,
		}
		if h.Required {
			item.Blocker = fmt.Sprintf("HTTP endpoint %s did not return %d", h.Name, h.ExpectedStatus)
		}
		r.Add(item)
	}

	for _, t := range p.TLS {
		checkTLSResult(r, t, tlsThreshold(cfg, t.Name))
	}

	for _, d := range p.DB {
		checkDBResult(r, d)
	}

	checkProxyDrift(r, snap)
}

// checkDBResult folds a single database wire-protocol probe result into the
// report. Required failures are Critical; unreachable but optional probes
// are Warnings. Reachable+not-handshake-ok (e.g. Redis returns -NOAUTH) is
// treated as a Warning regardless of Required — the server is up, the user
// just needs to configure auth.
func checkDBResult(r *report.Report, d model.DBProbeResult) {
	if d.Reachable && d.HandshakeOK {
		return
	}
	if d.Reachable && !d.HandshakeOK {
		// Server is up but didn't accept our ping.
		r.Add(report.Item{
			Category: "Database", Name: d.Name,
			Found: fmt.Sprintf("%s on %s:%d replied but handshake incomplete", d.Backend, d.Host, d.Port),
			Expected: fmt.Sprintf("%s server responds to unauthenticated probe", d.Backend),
			Severity: report.Warning,
		})
		return
	}
	sev := report.Warning
	var blocker string
	if d.Required {
		sev = report.Critical
		blocker = fmt.Sprintf("%s server %s (%s:%d) is unreachable", d.Backend, d.Name, d.Host, d.Port)
	}
	r.Add(report.Item{
		Category: "Database", Name: d.Name,
		Found:    fmt.Sprintf("%s:%d unreachable", d.Host, d.Port),
		Expected: fmt.Sprintf("%s server reachable", d.Backend),
		Severity: sev, Blocker: blocker,
	})
}

// tlsThreshold returns the warn_if_expires_within_days for an endpoint by name.
func tlsThreshold(cfg config.Config, name string) int {
	for _, ep := range cfg.TLS.Endpoints {
		if ep.Name == name {
			return ep.WarnIfExpiresWithinDays
		}
	}
	return 0
}

func checkTLSResult(r *report.Report, t model.TLSProbeResult, warnWithinDays int) {
	switch {
	case !t.HandshakeOK:
		sev := report.Warning
		if t.Required {
			sev = report.Critical
		}
		r.Add(report.Item{
			Category: "TLS", Name: t.Name,
			Found: "handshake failed", Expected: "handshake ok", Severity: sev,
			Blocker: blockerIf(t.Required, fmt.Sprintf("TLS handshake to %s failed", t.Name)),
		})
	case !t.HostnameOK:
		sev := report.Warning
		if t.Required {
			sev = report.Critical
		}
		r.Add(report.Item{
			Category: "TLS", Name: t.Name,
			Found: "hostname verification failed", Expected: "valid certificate", Severity: sev,
			Blocker: blockerIf(t.Required, fmt.Sprintf("TLS hostname verification for %s failed", t.Name)),
		})
	case t.Expired:
		r.Add(report.Item{
			Category: "TLS", Name: t.Name,
			Found: "certificate expired", Expected: "valid certificate", Severity: report.Critical,
			Blocker: fmt.Sprintf("TLS certificate for %s is expired", t.Name),
		})
	default:
		if warnWithinDays > 0 && t.DaysToExpiry <= warnWithinDays {
			r.Add(report.Item{
				Category: "TLS", Name: t.Name,
				Found:    fmt.Sprintf("expires in %d days", t.DaysToExpiry),
				Expected: fmt.Sprintf(">%d days", warnWithinDays),
				Severity: report.Warning,
			})
		}
	}
}

func blockerIf(required bool, msg string) string {
	if required {
		return msg
	}
	return ""
}

// checkProxyDrift is a placeholder hook; proxy-vs-baseline drift is evaluated
// in baseline comparison. In config-check mode there's no baseline reference,
// so nothing is emitted here.
func checkProxyDrift(_ *report.Report, _ model.Snapshot) {}
