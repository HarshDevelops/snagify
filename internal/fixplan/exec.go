package fixplan

import (
	"fmt"
	"io"
	"os/exec"
)

// runCmd runs a command and streams stdout/stderr to w.
func runCmd(w io.Writer, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = w
	cmd.Stderr = w
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %v: %w", name, args, err)
	}
	return nil
}
