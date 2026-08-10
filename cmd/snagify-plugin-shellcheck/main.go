// Command snagify-plugin-shellcheck is the canonical built-in Snagify plugin.
//
// It implements `kind: checks` of the snagify/plugin/v1 contract by simply
// running each Spec's command via os/exec and reporting the exit code.
// This file lives in `cmd/snagify-plugin-shellcheck` as a second main
// package so it can be built as `snagify-plugin-shellcheck` and dropped on
// $PATH. The host (snagify) does NOT shell into this — they communicate
// over a JSON-RPC pipe defined in internal/plugin.
//
// Build: go build -o snagify-plugin-shellcheck ./cmd/snagify-plugin-shellcheck
// Usage: snagify-plugin-shellcheck < /dev/stdin  (host writes JSON to stdin)
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	plugin "github.com/harshdevelops/snagify/internal/plugin"
)

func main() {
	req, err := plugin.DecodeRequest(os.Stdin)
	if err != nil {
		respondError(err)
		os.Exit(2)
	}

	results := make([]plugin.Result, 0, len(req.Checks))
	for _, c := range req.Checks {
		results = append(results, runOne(c))
	}

	out, err := plugin.EncodeResponse(results)
	if err != nil {
		fmt.Fprintln(os.Stderr, "encode:", err)
		os.Exit(2)
	}
	if _, err := os.Stdout.Write(out); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(2)
	}
	// Newline after the JSON object so shells / pipes don't get a partial
	// last line.
	_, _ = os.Stdout.Write([]byte("\n"))
}

func runOne(c plugin.Spec) plugin.Result {
	r := plugin.Result{Name: c.Name}
	if len(c.Run) == 0 || c.Run[0] == "" {
		r.Status = "warning"
		r.Message = "empty run command"
		return r
	}
	timeout := 30 * time.Second
	if c.Timeout != "" {
		if d, err := time.ParseDuration(c.Timeout); err == nil && d > 0 {
			timeout = d
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, c.Run[0], c.Run[1:]...)
	if c.Workdir != "" {
		cmd.Dir = c.Workdir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			r.ExitCode = exitErr.ExitCode()
		}
		if ctx.Err() == context.DeadlineExceeded {
			r.Status = "critical"
			r.Message = fmt.Sprintf("timed out after %s; output=%s", timeout, string(out))
			return r
		}
		if severityCritical(c.Severity) {
			r.Status = "critical"
		} else {
			r.Status = "warning"
		}
		if r.ExitCode != 0 {
			r.Message = fmt.Sprintf("exit %d: %s", r.ExitCode, truncate(string(out), 4096))
		} else {
			r.Message = truncate(string(out), 4096)
		}
		return r
	}
	r.Status = "ok"
	return r
}

func severityCritical(s string) bool {
	return s == "critical"
}

func truncate(s string, n int) string {
	if len(s) > n {
		return s[:n] + "...(truncated)"
	}
	return s
}

func respondError(err error) {
	resp := plugin.Response{Protocol: plugin.Protocol, Error: err.Error()}
	out, mErr := json.Marshal(resp)
	if mErr != nil {
		_, _ = io.WriteString(os.Stderr, mErr.Error())
		return
	}
	_, _ = os.Stdout.Write(out)
	_, _ = os.Stdout.Write([]byte("\n"))
}
