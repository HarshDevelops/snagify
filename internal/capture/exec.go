// Package capture gathers environment, runtime, service, and project state
// from the current machine into a model.Snapshot.
package capture

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

// cmdTimeout bounds how long any probe command may run.
const cmdTimeout = 5 * time.Second

// runCmd executes name with args under a timeout and returns combined,
// trimmed output plus whether it succeeded.
func runCmd(name string, args ...string) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), cmdTimeout)
	defer cancel()

	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return "", false
	}
	if err != nil {
		// Some tools (e.g. `java -version`) write version to stderr and
		// still exit 0; others exit non-zero but print usable output.
		text := strings.TrimSpace(string(out))
		if text == "" {
			return "", false
		}
		return text, true
	}
	return strings.TrimSpace(string(out)), true
}

// lookPath returns the resolved binary path, or "" if not found.
func lookPath(name string) string {
	p, err := exec.LookPath(name)
	if err != nil {
		return ""
	}
	return p
}
