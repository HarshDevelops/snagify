package check

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/report"
)

func writeDotEnv(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func envConfig(required, recommended, optional []string) config.Config {
	cfg := baseConfig()
	cfg.Env = config.Env{
		ActualFile:  ".env",
		Required:    required,
		Recommended: recommended,
		Optional:    optional,
	}
	return cfg
}

func TestCheckEnv_RequiredMissingIsCritical(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "OTHER_KEY=value\n")
	cfg := envConfig([]string{"DATABASE_URL", "JWT_SECRET"}, nil, nil)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	if !r.Failed() {
		t.Fatal("expected critical when required keys missing")
	}
	crits := r.Critical()
	if len(crits) != 1 || crits[0].Category != "Env" {
		t.Fatalf("expected 1 grouped env critical, got %d: %+v", len(crits), crits)
	}
	// Blocker should mention both keys.
	blocker := crits[0].Blocker
	if !strings.Contains(blocker, "DATABASE_URL") || !strings.Contains(blocker, "JWT_SECRET") {
		t.Errorf("blocker should list both missing keys, got: %s", blocker)
	}
}

func TestCheckEnv_RequiredPresentPasses(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "DATABASE_URL=x\nJWT_SECRET=y\n")
	cfg := envConfig([]string{"DATABASE_URL", "JWT_SECRET"}, nil, nil)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	if r.Failed() {
		t.Fatalf("expected pass when all required present: %v", r.Blockers())
	}
}

func TestCheckEnv_RecommendedMissingIsWarning(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "DATABASE_URL=x\n") // recommended keys absent
	cfg := envConfig(
		[]string{"DATABASE_URL"},
		[]string{"MAX_RETRIES", "TIMEOUT_MS"},
		nil,
	)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	if r.Failed() {
		t.Errorf("recommended missing should not fail: %v", r.Blockers())
	}
	if len(r.Warnings()) == 0 {
		t.Error("expected a warning for missing recommended keys")
	}
}

func TestCheckEnv_OptionalMissingIsSilent(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "DATABASE_URL=x\n")
	cfg := envConfig(
		[]string{"DATABASE_URL"},
		nil,
		[]string{"FALLBACK_1_API_KEY", "FALLBACK_2_API_KEY"},
	)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	if r.Failed() {
		t.Errorf("optional missing should not fail: %v", r.Blockers())
	}
	for _, it := range r.Items {
		if it.Category == "Env" {
			t.Errorf("optional missing should not appear in report: %+v", it)
		}
	}
}

func TestCheckEnv_GroupedOutputInTerminal(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "")
	cfg := envConfig(
		[]string{"DATABASE_URL", "JWT_SECRET", "API_KEY"},
		[]string{"MAX_RETRIES"},
		nil,
	)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	var buf bytes.Buffer
	report.Terminal(&buf, r)
	out := buf.String()

	// Should NOT have 3 separate rows for 3 keys — should be grouped.
	rows := strings.Count(out, ".env")
	if rows > 2 { // 1 critical row + maybe 1 recommended row
		t.Errorf("expected grouped output (<=2 .env rows), got %d", rows)
	}
	// All three missing keys should appear somewhere in the blockers line.
	if !strings.Contains(out, "DATABASE_URL") || !strings.Contains(out, "JWT_SECRET") {
		t.Errorf("expected missing keys named in output:\n%s", out)
	}
}

func TestCheckEnv_GroupedOutputInMarkdown(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "")
	cfg := envConfig(
		[]string{"DATABASE_URL", "JWT_SECRET"},
		nil,
		nil,
	)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	var buf bytes.Buffer
	report.Markdown(&buf, r)
	out := buf.String()

	// Should not produce 2 separate table rows, just 1 grouped one.
	rows := strings.Count(out, "| .env |")
	if rows > 1 {
		t.Errorf("markdown should group .env into 1 row, got %d", rows)
	}
}

func TestCheckEnv_JSONEnvSummary(t *testing.T) {
	dir := t.TempDir()
	writeDotEnv(t, dir, ".env", "DATABASE_URL=x\n")
	cfg := envConfig(
		[]string{"DATABASE_URL", "JWT_SECRET"},
		[]string{"MAX_RETRIES"},
		nil,
	)

	r := Run(baseSnapshot(), cfg, dir, Options{Probe: allFree})
	var buf bytes.Buffer
	if err := report.JSON(&buf, r); err != nil {
		t.Fatalf("JSON: %v", err)
	}

	// Must be parseable.
	var d map[string]any
	if err := json.Unmarshal(buf.Bytes(), &d); err != nil {
		t.Fatalf("JSON not parseable: %v", err)
	}
	// Check that the blocker text contains the missing key.
	raw, _ := json.Marshal(d)
	if !strings.Contains(string(raw), "JWT_SECRET") {
		t.Error("expected JWT_SECRET to appear in JSON output")
	}
}
