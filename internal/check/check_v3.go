package check

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/report"
)

// checkGit validates branch, clean, and dirty rules.
func checkGit(r *report.Report, snap model.Snapshot, cfg config.Config) {
	g := snap.Git
	if g == nil {
		return
	}
	want := cfg.Git

	if !g.Present {
		if want.RequireBranch != "" || want.RequireClean {
			r.Add(report.Item{
				Category: "Git", Name: "repository",
				Found: "not a git repo", Expected: "git repository", Severity: report.Critical,
				Blocker: "project is not a git repository",
			})
		}
		return
	}

	if want.RequireBranch != "" && g.Branch != want.RequireBranch {
		r.Add(report.Item{
			Category: "Git", Name: "branch",
			Found: orEmpty(g.Branch), Expected: want.RequireBranch, Severity: report.Critical,
			Blocker: fmt.Sprintf("git branch is %q, required %q", g.Branch, want.RequireBranch),
		})
	}

	if want.RequireClean && (g.Dirty || g.Untracked) {
		r.Add(report.Item{
			Category: "Git", Name: "working tree",
			Found: "dirty", Expected: "clean", Severity: report.Critical,
			Blocker: "git working tree must be clean (uncommitted changes present)",
		})
	} else if want.WarnIfDirty && (g.Dirty || g.Untracked) {
		r.Add(report.Item{
			Category: "Git", Name: "working tree",
			Found: "dirty", Expected: "clean", Severity: report.Warning,
		})
	}
}

// checkPath validates required commands exist on PATH.
func checkPath(r *report.Report, snap model.Snapshot, cfg config.Config) {
	if snap.Path == nil {
		return
	}
	for _, cmd := range cfg.Path.RequireCommands {
		info, ok := snap.Path.Executables[cmd]
		if !ok || !info.Present {
			r.Add(report.Item{
				Category: "Path", Name: cmd,
				Found: "missing", Expected: "on PATH", Severity: report.Critical,
				Blocker: fmt.Sprintf("required command %q not found on PATH", cmd),
			})
		}
	}
}

// checkDocker validates docker requirement, compose files, images, containers.
func checkDocker(r *report.Report, snap model.Snapshot, cfg config.Config, projectRoot string) {
	dc := cfg.Docker
	if !dc.Required && len(dc.ComposeFiles) == 0 && len(dc.RequiredImages) == 0 && len(dc.RequiredContainers) == 0 {
		return
	}
	d := snap.Docker

	if dc.Required && (d == nil || !d.Runtime.Present) {
		r.Add(report.Item{
			Category: "Docker", Name: "runtime",
			Found: "missing", Expected: "installed", Severity: report.Critical,
			Blocker: "Docker is required but not installed",
		})
	}

	for _, cf := range dc.ComposeFiles {
		if !fileExists(filepath.Join(projectRoot, cf)) {
			r.Add(report.Item{
				Category: "Docker", Name: cf,
				Found: "missing", Expected: "present", Severity: report.Critical,
				Blocker: fmt.Sprintf("required compose file %q is missing", cf),
			})
		}
	}

	// Image and container checks only when docker info is available.
	if d == nil || !d.Runtime.Present {
		return
	}

	for _, want := range dc.RequiredImages {
		if !hasImage(d.Images, want) {
			r.Add(report.Item{
				Category: "Docker", Name: want,
				Found: "not pulled", Expected: "available", Severity: report.Warning,
			})
		}
	}
	for _, want := range dc.RequiredContainers {
		if !hasContainer(d.Containers, want) {
			r.Add(report.Item{
				Category: "Docker", Name: want,
				Found: "not running", Expected: "running", Severity: report.Critical,
				Blocker: fmt.Sprintf("required container %q is not running", want),
			})
		}
	}
}

// checkSystem validates timezone, locale, case-sensitivity, and arch.
func checkSystem(r *report.Report, snap model.Snapshot, cfg config.Config) {
	want := cfg.System
	s := snap.System

	if len(want.AllowedArch) > 0 {
		if !containsFoldSlice(want.AllowedArch, snap.Environment.Arch) {
			r.Add(report.Item{
				Category: "System", Name: "arch",
				Found: snap.Environment.Arch, Expected: strings.Join(want.AllowedArch, "|"),
				Severity: report.Warning,
			})
		}
	}

	if s == nil {
		return
	}

	if want.Timezone != "" && !strings.EqualFold(s.Timezone, want.Timezone) {
		r.Add(report.Item{
			Category: "System", Name: "timezone",
			Found: orEmpty(s.Timezone), Expected: want.Timezone, Severity: report.Warning,
		})
	}
	if want.Locale != "" && !strings.EqualFold(s.Locale, want.Locale) {
		r.Add(report.Item{
			Category: "System", Name: "locale",
			Found: orEmpty(s.Locale), Expected: want.Locale, Severity: report.Warning,
		})
	}
	if want.CaseSensitiveFS != nil && s.CaseSensitiveFS != nil && *want.CaseSensitiveFS != *s.CaseSensitiveFS {
		r.Add(report.Item{
			Category: "System", Name: "case_sensitive_fs",
			Found: boolStr(*s.CaseSensitiveFS), Expected: boolStr(*want.CaseSensitiveFS),
			Severity: report.Warning,
		})
	}
}

func hasImage(images []model.DockerImageInfo, want string) bool {
	repo, tag := splitImage(want)
	for _, img := range images {
		if img.Repository == repo && (tag == "" || img.Tag == tag) {
			return true
		}
	}
	return false
}

func splitImage(s string) (repo, tag string) {
	if i := strings.LastIndex(s, ":"); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func hasContainer(containers []model.DockerContainerInfo, want string) bool {
	for _, c := range containers {
		if c.Name == want || strings.Contains(c.Name, want) || strings.HasPrefix(c.Image, want) {
			return true
		}
	}
	return false
}

func containsFoldSlice(list []string, v string) bool {
	for _, x := range list {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}

func orEmpty(s string) string {
	if s == "" {
		return "(unset)"
	}
	return s
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
