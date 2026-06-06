package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestCheckGit_BranchMismatchCritical(t *testing.T) {
	snap := baseSnapshot()
	snap.Git = &model.GitInfo{Present: true, Branch: "feature/x"}
	cfg := baseConfig()
	cfg.Git = config.Git{RequireBranch: "main"}

	r := Run(snap, cfg, t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected branch mismatch to be critical")
	}
}

func TestCheckGit_DirtyWithRequireClean(t *testing.T) {
	snap := baseSnapshot()
	snap.Git = &model.GitInfo{Present: true, Branch: "main", Dirty: true}
	cfg := baseConfig()
	cfg.Git = config.Git{RequireBranch: "main", RequireClean: true}

	r := Run(snap, cfg, t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected dirty tree to be critical with require_clean")
	}
}

func TestCheckGit_DirtyWarnOnly(t *testing.T) {
	snap := baseSnapshot()
	snap.Git = &model.GitInfo{Present: true, Branch: "main", Untracked: true}
	cfg := baseConfig()
	cfg.Git = config.Git{RequireBranch: "main", WarnIfDirty: true}

	r := Run(snap, cfg, t.TempDir(), Options{Probe: allFree})
	if r.Failed() {
		t.Errorf("warn_if_dirty should not fail the check: %v", r.Blockers())
	}
	if len(r.Warnings()) == 0 {
		t.Error("expected a dirty-tree warning")
	}
}

func TestCheckPath_RequiredCommandMissing(t *testing.T) {
	snap := baseSnapshot()
	snap.Path = &model.PathInfo{Executables: map[string]model.ExecInfo{
		"node": {Present: true, Path: "/usr/bin/node"},
		// java intentionally absent
	}}
	cfg := baseConfig()
	cfg.Path = config.Path{RequireCommands: []string{"node", "java"}}

	r := Run(snap, cfg, t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected missing required command to be critical")
	}
}

func TestCheckDocker_RequiredMissingAndComposeMissing(t *testing.T) {
	snap := baseSnapshot()
	snap.Docker = &model.DockerInfo{Runtime: model.VersionInfo{Present: false}}
	cfg := baseConfig()
	cfg.Docker = config.Docker{Required: true, ComposeFiles: []string{"docker-compose.yml"}}

	r := Run(snap, cfg, t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected docker-required + missing compose to be critical")
	}
	// Two critical items: runtime missing and compose file missing.
	if len(r.Critical()) < 2 {
		t.Errorf("expected >=2 critical docker items, got %d", len(r.Critical()))
	}
}

func TestCheckDocker_ComposeFilePresentPasses(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "docker-compose.yml", "services: {}\n")

	snap := baseSnapshot()
	snap.Docker = &model.DockerInfo{Runtime: model.VersionInfo{Present: true, Version: "25.0.3"}}
	cfg := baseConfig()
	cfg.Docker = config.Docker{Required: true, ComposeFiles: []string{"docker-compose.yml"}}

	r := Run(snap, cfg, dir, Options{Probe: allFree})
	for _, it := range r.Critical() {
		if it.Category == "Docker" {
			t.Errorf("unexpected docker critical: %+v", it)
		}
	}
}

func TestCheckSystem_TimezoneAndCaseWarnings(t *testing.T) {
	falseVal := false
	trueVal := true
	snap := baseSnapshot()
	snap.System = &model.SystemInfo{Timezone: "UTC", Locale: "C", CaseSensitiveFS: &trueVal}
	cfg := baseConfig()
	cfg.System = config.System{Timezone: "Asia/Kolkata", CaseSensitiveFS: &falseVal}

	r := Run(snap, cfg, t.TempDir(), Options{Probe: allFree})
	if r.Failed() {
		t.Errorf("system mismatches should be warnings, not critical: %v", r.Blockers())
	}
	if len(r.Warnings()) < 2 {
		t.Errorf("expected timezone + case warnings, got %d", len(r.Warnings()))
	}
}

func TestCheckProbes_RequiredServiceUnreachableCritical(t *testing.T) {
	snap := baseSnapshot()
	snap.Probes = &model.ProbeResults{
		Ran: true,
		TCP: []model.TCPProbeResult{
			{Name: "postgres", Host: "localhost", Port: 5432, Required: true, Reachable: false},
			{Name: "es", Host: "localhost", Port: 9200, Required: false, Reachable: false},
		},
	}
	r := Run(snap, baseConfig(), t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected required unreachable service to be critical")
	}
	// Optional unreachable should be a warning, not critical.
	if len(r.Warnings()) == 0 {
		t.Error("expected optional service warning")
	}
}

func TestCheckProbes_HTTPWrongStatus(t *testing.T) {
	snap := baseSnapshot()
	snap.Probes = &model.ProbeResults{
		Ran:  true,
		HTTP: []model.HTTPProbeResult{{Name: "api", URL: "https://x", ExpectedStatus: 200, GotStatus: 500, Required: true, OK: false}},
	}
	r := Run(snap, baseConfig(), t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected required HTTP wrong-status to be critical")
	}
}

func TestCheckProbes_TLSExpiredCritical(t *testing.T) {
	snap := baseSnapshot()
	snap.Probes = &model.ProbeResults{
		Ran: true,
		TLS: []model.TLSProbeResult{{Name: "api", Host: "x", Required: true, HandshakeOK: true, HostnameOK: true, Expired: true}},
	}
	r := Run(snap, baseConfig(), t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected expired cert to be critical")
	}
}
