// Package customcheck executes user-defined shell checks declared in a
// project's .snagify.yaml. Custom checks fill the gap between Snagify's
// built-in runtimes/services/database probes and project-specific business
// invariants ("is the migration directory well-formed? does the script
// actually exit 0?"). Both the schema and the executor here are reused by
// the built-in snagify-plugin-shellcheck plugin in cmd/snagify-plugin-shellcheck.
package customcheck

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"time"
)

// Severity is reported in the same vocabulary Snagify uses elsewhere.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityWarning   Severity = "warning"
)

// Spec is one custom check, parsed from a `.snagify.yaml` custom-checks
// block. Marshal/Unmarshal are handled by gopkg.in/yaml.v3 once this type
// is embedded into config.Config.
type Spec struct {
	Name     string    `yaml:"name"`
	Run      []string  `yaml:"run"`
	Severity Severity  `yaml:"severity"`
	Message  string    `yaml:"message"`
	Timeout  string    `yaml:"timeout"`
	Workdir  string    `yaml:"workdir"`
}

// Result is one check's outcome. ExitCode=0 (and Error=="") means pass.
// ExitCode!=0 or Error!="" means fail; the Blockers convey *why*. Stdout
// and Stderr are surfaced (truncated) so the report is actionable but not
// gigabytes long.
type Result struct {
	Name      string
	Severity  Severity
	ExitCode  int
	Stdout    string
	Stderr    string
	Error     string
	Elapsed   time.Duration
	Critical  bool
}

// DefaultTimeout caps any user-supplied timeout that is missing or absurd.
const DefaultTimeout = 30 * time.Second

// MaxOutputBytes is the truncation limit for stdout/stderr captured into
// the report.
const MaxOutputBytes = 4096

// Run executes spec under its (resolved) timeout. The first element of
// spec.Run must resolve to a real binary; spec.Run[1:] are its arguments.
// The function returns a Result that always has Name and Severity set so
// callers don't have to nil-check.
func Run(spec Spec) Result {
	r := Result{Name: spec.Name, Severity: spec.Severity}
	if spec.Name == "" {
		r.Error = "custom check has empty name"
		r.Critical = true
		return r
	}
	if len(spec.Run) == 0 || spec.Run[0] == "" {
		r.Error = "custom check has empty run command"
		r.Critical = true
		return r
	}
	if spec.Severity != SeverityCritical && spec.Severity != SeverityWarning {
		// Default missing/unknown → warning (less likely to fail CI on typos).
		r.Severity = SeverityWarning
	}

	timeout := DefaultTimeout
	if spec.Timeout != "" {
		if d, err := time.ParseDuration(spec.Timeout); err == nil && d > 0 && d < 10*time.Minute {
			timeout = d
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, spec.Run[0], spec.Run[1:]...)
	if spec.Workdir != "" {
		cmd.Dir = spec.Workdir
	}

	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf

	start := time.Now()
	err := cmd.Run()
	r.Elapsed = time.Since(start)

	r.Stdout = truncate(outBuf.String())
	r.Stderr = truncate(errBuf.String())
	if err != nil {
		var exitErr *exec.ExitError
		switch {
		case errors.As(err, &exitErr):
			r.ExitCode = exitErr.ExitCode()
			// exec.CommandContext kills the child with SIGKILL when the
			// context expires, so on Unix the resulting error reports a
			// negative exit rather than a clean DeadlineExceeded. We treat
			// both shapes as a timeout.
			if ctx.Err() == context.DeadlineExceeded || (r.Elapsed >= timeout && r.ExitCode < 0) {
				r.Error = fmt.Sprintf("timed out after %s", timeout)
			} else {
				r.Error = fmt.Sprintf("exit %d", r.ExitCode)
			}
		case errors.Is(err, context.DeadlineExceeded):
			r.ExitCode = -1
			r.Error = fmt.Sprintf("timed out after %s", timeout)
		default:
			r.ExitCode = -1
			r.Error = err.Error()
		}
	}
	r.Critical = r.Severity == SeverityCritical
	return r
}

func truncate(s string) string {
	if len(s) > MaxOutputBytes {
		return s[:MaxOutputBytes] + "...(truncated)"
	}
	return s
}
