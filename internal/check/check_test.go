package check

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
)

func presentVI(v string) model.VersionInfo { return model.VersionInfo{Version: v, Present: true} }

func baseSnapshot() model.Snapshot {
	return model.Snapshot{
		Project: model.ProjectInfo{Name: "checkout-service"},
		Runtimes: model.Runtimes{
			Node:   presentVI("20.11.1"),
			Java:   presentVI("17.0.9"),
			Maven:  presentVI("3.9.6"),
			Docker: presentVI("25.0.3"),
		},
	}
}

func baseConfig() config.Config {
	return config.Config{
		Version:  1,
		Project:  config.Project{Name: "checkout-service"},
		Runtimes: map[string]string{"node": ">=20 <23", "java": "17", "maven": ">=3.8", "docker": "required"},
	}
}

// allFree treats every port as free (nothing listening).
func allFree(int) bool { return false }

func TestCheck_Passes(t *testing.T) {
	r := Run(baseSnapshot(), baseConfig(), t.TempDir(), Options{Probe: allFree})
	if r.Failed() {
		t.Fatalf("expected pass, got blockers: %v", r.Blockers())
	}
	if r.Status != "Your setup is ready" {
		t.Errorf("status = %q", r.Status)
	}
}

func TestCheck_FailsWhenRuntimeMissing(t *testing.T) {
	snap := baseSnapshot()
	snap.Runtimes.Maven = model.VersionInfo{Present: false}

	r := Run(snap, baseConfig(), t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected failure when maven missing")
	}
	if !hasBlockerContaining(r.Blockers(), "maven is missing") {
		t.Errorf("expected 'maven is missing' blocker, got %v", r.Blockers())
	}
}

func TestCheck_FailsWhenVersionMismatch(t *testing.T) {
	snap := baseSnapshot()
	snap.Runtimes.Java = presentVI("21.0.2")

	r := Run(snap, baseConfig(), t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected failure when java major mismatches")
	}
}

func TestCheck_FailsWhenEnvKeyMissing(t *testing.T) {
	dir := t.TempDir()
	// .env declares only DATABASE_URL; JWT_SECRET and REDIS_URL are missing.
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("DATABASE_URL=x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := baseConfig()
	cfg.Env = config.Env{ActualFile: ".env", Required: []string{"DATABASE_URL", "JWT_SECRET", "REDIS_URL"}}

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected failure when env keys missing")
	}
	if !hasBlockerContaining(r.Blockers(), "JWT_SECRET") {
		t.Errorf("expected JWT_SECRET blocker, got %v", r.Blockers())
	}
}

func TestCheck_FailsWhenPortNotListening(t *testing.T) {
	cfg := baseConfig()
	cfg.Ports = config.Ports{MustBeListening: []int{5432}}

	r := Run(baseSnapshot(), cfg, t.TempDir(), Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected failure when required port not listening")
	}
}

func TestCheck_FailsWhenPortOccupied(t *testing.T) {
	cfg := baseConfig()
	cfg.Ports = config.Ports{MustBeFree: []int{8080}}
	// Probe reports 8080 as listening (occupied).
	probe := func(p int) bool { return p == 8080 }

	r := Run(baseSnapshot(), cfg, t.TempDir(), Options{Probe: probe})
	if !r.Failed() {
		t.Fatal("expected failure when must-be-free port is occupied")
	}
}

func TestCheck_IgnoredRuntimeSkipped(t *testing.T) {
	cfg := baseConfig()
	cfg.Runtimes["rust"] = ">=1.70"
	cfg.Ignore.Runtimes = []string{"rust"}
	// Snapshot has no rust; if not ignored this would fail.
	r := Run(baseSnapshot(), cfg, t.TempDir(), Options{Probe: allFree})
	if r.Failed() {
		t.Fatalf("ignored rust should not cause failure: %v", r.Blockers())
	}
}

func hasBlockerContaining(blockers []string, sub string) bool {
	for _, b := range blockers {
		if containsFold(b, sub) {
			return true
		}
	}
	return false
}

func containsFold(s, sub string) bool {
	return len(sub) == 0 || indexFold(s, sub) >= 0
}

func indexFold(s, sub string) int {
	ls, lsub := toLower(s), toLower(sub)
	for i := 0; i+len(lsub) <= len(ls); i++ {
		if ls[i:i+len(lsub)] == lsub {
			return i
		}
	}
	return -1
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if 'A' <= c && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	return string(b)
}
