// Package check compares a captured snapshot against a .snagify.yaml config
// and produces a ranked report.
package check

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/report"
	"github.com/harshdevelops/snagify/internal/verreq"
)

// PortProbe checks whether a port is listening. Injectable for tests.
type PortProbe func(port int) bool

// Options configures a check run.
type Options struct {
	// Probe overrides port listening detection (defaults to capture).
	Probe PortProbe
}

// Run compares snap against cfg and returns a ranked report. projectRoot is
// used to resolve relative env file paths. If snap.Probes is populated (the
// caller ran active probes), those results are folded into the report.
func Run(snap model.Snapshot, cfg config.Config, projectRoot string, opts Options) report.Report {
	probe := opts.Probe
	if probe == nil {
		probe = capture.IsPortListening
	}

	name := cfg.Project.Name
	if name == "" {
		name = snap.Project.Name
	}
	r := report.Report{
		Project:    name,
		FoundLabel: "found",
		ExpLabel:   "expected",
	}

	checkRuntimes(&r, snap, cfg)
	checkEnv(&r, cfg, projectRoot)
	checkPorts(&r, cfg, probe)
	checkGit(&r, snap, cfg)
	checkPath(&r, snap, cfg)
	checkDocker(&r, snap, cfg, projectRoot)
	checkSystem(&r, snap, cfg)
	checkProbes(&r, snap, cfg)

	if r.Failed() {
		r.Status = "Your setup is not ready"
	} else {
		r.Status = "Your setup is ready"
	}
	return r
}

func checkRuntimes(r *report.Report, snap model.Snapshot, cfg config.Config) {
	names := make([]string, 0, len(cfg.Runtimes))
	for n := range cfg.Runtimes {
		names = append(names, n)
	}
	sort.Strings(names)

	for _, rtName := range names {
		if cfg.IgnoredRuntime(rtName) {
			continue
		}
		req := cfg.Runtimes[rtName]
		info, known := model.LookupRuntime(snap.Runtimes, rtName)
		if !known {
			// Unknown runtime key in config; surface as a warning rather than
			// crashing or silently ignoring.
			r.Add(report.Item{
				Category: "Runtime", Name: rtName,
				Found: "unknown runtime", Expected: req, Severity: report.Warning,
			})
			continue
		}

		if verreq.IsRequiredKeyword(req) {
			if !info.Present {
				r.Add(report.Item{
					Category: "Runtime", Name: rtName,
					Found: "missing", Expected: "required", Severity: report.Critical,
					Blocker: fmt.Sprintf("%s is missing", rtName),
				})
			}
			continue
		}

		if !info.Present {
			r.Add(report.Item{
				Category: "Runtime", Name: rtName,
				Found: "missing", Expected: req, Severity: report.Critical,
				Blocker: fmt.Sprintf("%s is missing", rtName),
			})
			continue
		}

		res := verreq.Check(info.Version, req)
		switch {
		case res.OK:
			// satisfied; nothing to report
		case !res.Parsed:
			r.Add(report.Item{
				Category: "Runtime", Name: rtName,
				Found: info.Version, Expected: req, Severity: report.Warning,
				Blocker: fmt.Sprintf("%s version %q could not be verified against %q", rtName, info.Version, req),
			})
		default:
			r.Add(report.Item{
				Category: "Runtime", Name: rtName,
				Found: info.Version, Expected: req, Severity: report.Critical,
				Blocker: fmt.Sprintf("%s version does not match requirement (found %s, expected %s)", rtName, info.Version, req),
			})
		}
	}
}

func checkEnv(r *report.Report, cfg config.Config, projectRoot string) {
	if len(cfg.Env.Required) == 0 && len(cfg.Env.Recommended) == 0 {
		return
	}
	actual := cfg.Env.ActualFile
	if actual == "" {
		actual = ".env"
	}
	if !filepath.IsAbs(actual) {
		actual = filepath.Join(projectRoot, actual)
	}

	keys, exists := capture.ReadEnvKeys(actual)
	present := map[string]bool{}
	for _, k := range keys {
		present[k] = true
	}

	// Required missing => grouped critical.
	var missingRequired []string
	for _, req := range cfg.Env.Required {
		if !exists || !present[req] {
			missingRequired = append(missingRequired, req)
		}
	}
	if len(missingRequired) > 0 {
		joined := strings.Join(missingRequired, ", ")
		n := len(missingRequired)
		label := fmt.Sprintf("%d required %s missing", n, plural("key", n))
		r.Add(report.Item{
			Category: "Env", Name: ".env",
			Found:    label,
			Expected: "all present",
			Severity: report.Critical,
			Blocker:  fmt.Sprintf("Required env keys are absent: %s", joined),
		})
	}

	// Recommended missing => grouped warning.
	var missingRecommended []string
	for _, rec := range cfg.Env.Recommended {
		if !exists || !present[rec] {
			missingRecommended = append(missingRecommended, rec)
		}
	}
	if len(missingRecommended) > 0 {
		n := len(missingRecommended)
		label := fmt.Sprintf("%d recommended %s missing", n, plural("key", n))
		joined := strings.Join(missingRecommended, ", ")
		r.Add(report.Item{
			Category: "Env", Name: ".env",
			Found:    label,
			Expected: "present (recommended)",
			Severity: report.Warning,
			Blocker:  fmt.Sprintf("Recommended env keys are absent: %s", joined),
		})
	}
	// Optional missing => silently ignored.
}

func plural(word string, n int) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func checkPorts(r *report.Report, cfg config.Config, probe PortProbe) {
	for _, p := range cfg.Ports.MustBeListening {
		if !probe(p) {
			r.Add(report.Item{
				Category: "Port", Name: fmt.Sprintf("%d", p),
				Found: "not listening", Expected: "listening", Severity: report.Critical,
				Blocker: fmt.Sprintf("port %d is not listening", p),
			})
		}
	}
	for _, p := range cfg.Ports.MustBeFree {
		if probe(p) {
			r.Add(report.Item{
				Category: "Port", Name: fmt.Sprintf("%d", p),
				Found: "occupied", Expected: "free", Severity: report.Critical,
				Blocker: fmt.Sprintf("port %d is already occupied", p),
			})
		}
	}
}

// FindConfig locates a .snagify.yaml: explicit path if given, else config.FileName
// in projectRoot. Returns the resolved path and whether it exists.
func FindConfig(explicit, projectRoot string) (string, bool) {
	if explicit != "" {
		return explicit, fileExists(explicit)
	}
	p := filepath.Join(projectRoot, config.FileName)
	return p, fileExists(p)
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
