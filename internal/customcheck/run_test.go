package customcheck

import (
	"strings"
	"testing"
	"time"
)

func whichExists(t *testing.T, bin string) bool {
	t.Helper()
	for _, cand := range []string{bin, "/usr/bin/" + bin, "/bin/" + bin} {
		if _, err := execLookPath(cand); err == nil {
			return true
		}
	}
	return false
}

func TestRun_ExitZero(t *testing.T) {
	if !whichExists(t, "true") {
		t.Skip("`true` not on PATH")
	}
	res := Run(Spec{Name: "is-ok", Run: []string{"true"}, Severity: SeverityCritical})
	if res.Error != "" {
		t.Fatalf("expected no error, got %q", res.Error)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit 0, got %d", res.ExitCode)
	}
	if !res.Critical {
		t.Errorf("expected Critical=true for severity=critical")
	}
}

func TestRun_ExitNonZero(t *testing.T) {
	if !whichExists(t, "false") {
		t.Skip("`false` not on PATH")
	}
	res := Run(Spec{Name: "should-fail", Run: []string{"false"}, Severity: SeverityWarning})
	if res.Error == "" {
		t.Fatalf("expected an error description")
	}
	if res.ExitCode != 1 {
		// `false` may not be installed on Windows; skip then.
		t.Logf("got exit=%d (may be normal on Windows)", res.ExitCode)
	}
	if res.Critical {
		t.Errorf("expected Critical=false for severity=warning")
	}
}

func TestRun_CapturesStdout(t *testing.T) {
	if !whichExists(t, "echo") {
		t.Skip("`echo` not on PATH")
	}
	res := Run(Spec{Name: "prints", Run: []string{"echo", "hello-there"}, Severity: SeverityWarning})
	if !strings.Contains(res.Stdout, "hello-there") {
		t.Errorf("expected stdout capture, got %q", res.Stdout)
	}
}

func TestRun_Timeout(t *testing.T) {
	if !whichExists(t, "sleep") {
		t.Skip("`sleep` not on PATH")
	}
	res := Run(Spec{
		Name:    "slow",
		Run:     []string{"sleep", "10"},
		Timeout: "100ms",
	})
	if res.Error == "" || !strings.Contains(strings.ToLower(res.Error), "timed out") {
		t.Errorf("expected timeout error, got %q", res.Error)
	}
	if res.Elapsed > 2*time.Second {
		t.Errorf("timeout did not fire fast enough: %s", res.Elapsed)
	}
}

func TestRun_EmptyName(t *testing.T) {
	res := Run(Spec{Name: "", Run: []string{"true"}})
	if res.Error == "" {
		t.Errorf("expected error for empty name")
	}
	if !res.Critical {
		t.Errorf("empty name should be treated as critical")
	}
}

func TestRun_EmptyRun(t *testing.T) {
	res := Run(Spec{Name: "x"})
	if res.Error == "" {
		t.Errorf("expected error for empty run")
	}
}

func TestRun_DefaultsUnknownSeverityToWarning(t *testing.T) {
	if !whichExists(t, "true") {
		t.Skip("`true` not on PATH")
	}
	res := Run(Spec{Name: "x", Run: []string{"true"}, Severity: "weird"})
	if res.Severity != SeverityWarning {
		t.Errorf("expected default to warning, got %v", res.Severity)
	}
	if res.Critical {
		t.Errorf("unknown severity must not be critical")
	}
}

func TestRun_BadTimeoutFallsBack(t *testing.T) {
	if !whichExists(t, "true") {
		t.Skip("`true` not on PATH")
	}
	res := Run(Spec{Name: "x", Run: []string{"true"}, Timeout: "not-a-duration"})
	if res.Error != "" {
		t.Errorf("bad timeout should fall back; got %q", res.Error)
	}
}
