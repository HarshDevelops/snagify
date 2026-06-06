// Package canary contains an end-to-end test asserting that a secret value in
// a .env file never appears in any snapshot, baseline, or rendered output.
package canary

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harshdevelops/snagify/internal/baseline"
	"github.com/harshdevelops/snagify/internal/capture"
	"github.com/harshdevelops/snagify/internal/check"
	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/report"
)

const secret = "THIS_SHOULD_NEVER_APPEAR_12345"

func TestSecretNeverLeaks(t *testing.T) {
	dir := t.TempDir()
	write(t, dir, "go.mod", "module example.com/app\n")
	write(t, dir, ".env.example", "DATABASE_URL=\nJWT_SECRET=\nREDIS_URL=\n")
	// DATABASE_URL carries the canary VALUE (must never leak). JWT_SECRET is
	// intentionally omitted so its KEY NAME surfaces as a missing key — proving
	// names may appear while values never do.
	write(t, dir, ".env", "DATABASE_URL="+secret+"\nREDIS_URL=redis://localhost:6379\n")

	snap, err := capture.Capture(capture.Options{ProjectRoot: dir})
	if err != nil {
		t.Fatalf("capture: %v", err)
	}

	// 1. Raw snapshot JSON
	assertNoSecret(t, "snapshot json", mustJSON(t, snap))

	// 2. Sanitized baseline JSON
	base := baseline.Sanitize(snap)
	assertNoSecret(t, "baseline json", mustJSON(t, base))

	// 3. Baseline comparison report in all formats
	rep := baseline.Compare(base, snap)
	assertReportFormatsClean(t, "baseline report", rep)

	// 4. Config check report in all formats
	cfg := config.Config{
		Project:  config.Project{Name: "app"},
		Runtimes: map[string]string{"go": "required"},
		Env:      config.Env{ActualFile: ".env", Required: []string{"DATABASE_URL", "JWT_SECRET", "REDIS_URL"}},
	}
	checkRep := check.Run(snap, cfg, dir, check.Options{Probe: func(int) bool { return false }})
	assertReportFormatsClean(t, "check report", checkRep)

	// Sanity: the env KEY name is allowed to appear; only the VALUE must not.
	repJSON := renderJSON(t, checkRep)
	if !strings.Contains(repJSON, "JWT_SECRET") {
		t.Error("expected env key name JWT_SECRET to be tracked (presence only)")
	}
}

func assertReportFormatsClean(t *testing.T, label string, r report.Report) {
	t.Helper()
	var term, md bytes.Buffer
	report.Terminal(&term, r)
	report.Markdown(&md, r)
	assertNoSecret(t, label+" terminal", term.String())
	assertNoSecret(t, label+" markdown", md.String())
	assertNoSecret(t, label+" json", renderJSON(t, r))
}

func renderJSON(t *testing.T, r report.Report) string {
	t.Helper()
	var buf bytes.Buffer
	if err := report.JSON(&buf, r); err != nil {
		t.Fatalf("render json: %v", err)
	}
	return buf.String()
}

func assertNoSecret(t *testing.T, where, content string) {
	t.Helper()
	if strings.Contains(content, secret) {
		t.Fatalf("secret value leaked in %s", where)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}
