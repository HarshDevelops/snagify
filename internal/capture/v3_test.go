package capture

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRedactHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no home dir")
	}
	in := filepath.Join(home, ".nvm", "versions", "node")
	got := redactHome(in)
	if !strings.HasPrefix(got, "~") {
		t.Errorf("expected redacted path to start with ~, got %q", got)
	}
	if strings.Contains(got, home) {
		t.Errorf("redacted path still contains home: %q", got)
	}
	// Unrelated path is unchanged.
	if redactHome("/usr/bin/node") != "/usr/bin/node" {
		t.Error("non-home path should be unchanged")
	}
}

func TestCaptureSystem_PopulatesFields(t *testing.T) {
	s := captureSystem()
	if s.Timezone == "" {
		t.Error("expected timezone to be populated")
	}
	// CaseSensitiveFS should be determinable on the test host.
	if s.CaseSensitiveFS == nil {
		t.Error("expected case-sensitivity to be determined")
	}
}

func TestDetectCaseSensitiveFS_NoLeftoverTempFiles(t *testing.T) {
	before := countSnagifyTemps(t)
	if _, ok := detectCaseSensitiveFS(); !ok {
		t.Skip("case sensitivity undeterminable")
	}
	after := countSnagifyTemps(t)
	if after > before {
		t.Errorf("temp files leaked: before=%d after=%d", before, after)
	}
}

func countSnagifyTemps(t *testing.T) int {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), "snagify-case") {
			n++
		}
	}
	return n
}

func TestCapturePath_RedactsAndResolves(t *testing.T) {
	p := capturePath()
	if p.Executables == nil {
		t.Fatal("executables map nil")
	}
	// git is almost always present in CI/dev; if so, path must be redacted.
	if info, ok := p.Executables["git"]; ok && info.Present {
		home, _ := os.UserHomeDir()
		if home != "" && strings.Contains(info.Path, home) {
			t.Errorf("git path not redacted: %q", info.Path)
		}
	}
}
