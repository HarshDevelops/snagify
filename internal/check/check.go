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
	"github.com/harshdevelops/snagify/internal/customcheck"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/plugin"
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
	checkEnvSafety(&r, snap)
	checkCustom(&r, cfg, projectRoot)
	checkPlugins(&r, cfg, projectRoot)

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

// checkEnvSafety folds the captured .env safety report into blockers.
// Tracking .env in git is critical: it almost always leaks production
// secrets. The other safety items (missing .gitignore, weak mode, weak
// secrets) are warnings, since fixing them doesn't block the project from
// running.
func checkEnvSafety(r *report.Report, snap model.Snapshot) {
	safety := snap.EnvFiles.Safety
	if safety == nil {
		return
	}
	if safety.Tracked {
		r.Add(report.Item{
			Category: "EnvSafety", Name: ".env (tracked)",
			Found: "tracked in git", Expected: "gitignored",
			Severity: report.Critical,
			Blocker: ".env is tracked in git; remove it from tracking and rotate any leaked secrets immediately",
		})
	}
	if safety.EnvFile != "" && !safety.Gitignored {
		r.Add(report.Item{
			Category: "EnvSafety", Name: ".env (gitignore)",
			Found: "no .gitignore rule covers it", Expected: "covered by .gitignore",
			Severity: report.Warning,
			Blocker: ".env exists but is not covered by any .gitignore; add '.env' (or '.env*') to .gitignore",
		})
	}
	if safety.WeakMode {
		r.Add(report.Item{
			Category: "EnvSafety", Name: ".env (mode)",
			Found: "weak permissions", Expected: "0600",
			Severity: report.Warning,
			Blocker: ".env is readable by group or other; chmod 0600",
		})
	}
	if len(safety.WeakSecrets) > 0 {
		var keys []string
		for _, h := range safety.WeakSecrets {
			keys = append(keys, h.Key)
		}
		sort.Strings(keys)
		r.Add(report.Item{
			Category: "EnvSafety", Name: ".env (weak secrets)",
			Found: fmt.Sprintf("%d leaked-looking %s", len(keys), plural("value", len(keys))),
			Expected: "no leaked secrets",
			Severity: report.Warning,
			Blocker: fmt.Sprintf(".env contains values matching known secret shapes (keys: %s); rotate them in your secret manager", strings.Join(keys, ", ")),
		})
	}
}

// checkCustom executes each `checks.custom` entry under its declared
// timeout, converting exit codes into report items of category "Custom".
// Snagify never auto-applies fixes for these — the `fix` action is the
// user's responsibility — so failures are reported but not actioned.
func checkCustom(r *report.Report, cfg config.Config, projectRoot string) {
	if len(cfg.Checks.Custom) == 0 {
		return
	}
	for _, c := range cfg.Checks.Custom {
		spec := customcheck.Spec{
			Name:     c.Name,
			Run:      c.Run,
			Severity: customcheck.Severity(c.Severity),
			Message:  c.Message,
			Timeout:  c.Timeout,
			Workdir:  c.Workdir,
		}
		if spec.Workdir == "" {
			spec.Workdir = projectRoot
		}
		res := customcheck.Run(spec)
		if res.Error == "" {
			continue
		}
		sev := report.Warning
		if res.Critical {
			sev = report.Critical
		}
		blocker := fmt.Sprintf("custom check %q failed: %s", c.Name, res.Error)
		if c.Message != "" {
			blocker = fmt.Sprintf("%s (%s)", c.Message, res.Error)
		}
		r.Add(report.Item{
			Category: "Custom", Name: c.Name,
			Found: res.Error, Expected: "exit 0",
			Severity: sev, Blocker: blocker,
		})
	}
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

// checkPlugins dispatches each configured plugin through the plugin
// loader. Plugins that refuse to start, exit non-zero, or send malformed
// JSON produce a Warning - never a Critical - because a broken plugin
// shouldn't fail the user's CI run on its own. Per-check results are
// translated into report items using the same severity as the plugin
// result.
func checkPlugins(r *report.Report, cfg config.Config, projectRoot string) {
	if len(cfg.Plugins.Items) == 0 {
		return
	}
	for _, p := range cfg.Plugins.Items {
		spec := plugin.PluginSpec{
			Name:        p.Name,
			Description: p.Description,
			Binary:      p.Binary,
			Command:     p.Command,
			Args:        p.Args,
			Timeout:     p.Timeout,
		}
		rr := plugin.Run(spec, nil, projectRoot)
		if rr.Error != "" {
			r.Add(report.Item{
				Category: "Plugin", Name: p.Name,
				Found: rr.Error, Expected: "plugin succeeded",
				Severity: report.Warning,
				Blocker: fmt.Sprintf("plugin %q: %s", p.Name, rr.Error),
			})
			continue
		}
		for _, res := range rr.Results {
			sev := report.Warning
			if res.Status == "critical" {
				sev = report.Critical
			} else if res.Status == "ok" {
				continue
			}
			r.Add(report.Item{
				Category: "Plugin", Name: res.Name,
				Found:    res.Message,
				Expected: "ok",
				Severity: sev,
				Blocker:  fmt.Sprintf("plugin %q reported %s for %q: %s", p.Name, res.Status, res.Name, res.Message),
			})
		}
	}
}
