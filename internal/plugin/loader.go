package plugin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DefaultTimeout is applied when a PluginSpec doesn't supply one.
const DefaultTimeout = 30 * time.Second

// DefaultExecTimeout caps the timeout at 5 minutes so a typo'd plugin
// can't wedge the host.
const DefaultExecTimeout = 5 * time.Minute

// loadLookPath is exposed to tests via an indirection.
var loadLookPath = exec.LookPath

// RunResult is what Snagify (the host) gets back from a plugin invocation.
// It is *not* the same as the wire Result — this is the loader's view
// (whole-plugin outcome + error).
type RunResult struct {
	// Spec is the spec that was sent; preserved so callers can correlate
	// report items back to config entries.
	Spec config_Spec
	// Results are the per-check outcomes from the plugin payload.
	Results []Result
	// Error is non-empty when the plugin could not be invoked, exited
	// non-zero, sent malformed JSON, or timed out. Per-check failures live
	// in the Results slice, never here.
	Error string
}

// config_Spec is a stand-in for config.PluginSpec to avoid an import cycle
// (config -> plugin is one-way; plugin can't import config). The wire-level
// Spec is what travels; the host's PluginSpec is what called us, and we
// echo it back inside RunResult so callers see what was executed.
type config_Spec struct {
	Name        string
	Description string
	Binary      string
	Command     string
	Args        []string
	Timeout     string
}

// PluginSpec is the looser type Run accepts from caller code. It mirrors
// config.PluginSpec but avoids the import.
type PluginSpec struct {
	Name        string
	Description string
	Binary      string
	Command     string
	Args        []string
	Timeout     string
}

// Run invokes a single plugin against a set of checks. The plugin is
// resolved in this order:
//  1. Spec.Command (split into argv, exec'd directly)
//  2. Spec.Binary (resolved via exec.LookPath)
//
// Either must be set. The plugin receives the JSON Request on stdin and
// must respond with the JSON Response on stdout; stderr is surfaced
// verbatim into the error path so plugin authors can debug.
func Run(spec PluginSpec, checks []Spec, workdir string) RunResult {
	rr := RunResult{Spec: config_Spec{
		Name:        spec.Name,
		Description: spec.Description,
		Binary:      spec.Binary,
		Command:     spec.Command,
		Args:        spec.Args,
		Timeout:     spec.Timeout,
	}}
	if spec.Name == "" {
		rr.Error = "plugin spec has empty name"
		return rr
	}

	argv, err := resolveCommand(spec)
	if err != nil {
		rr.Error = err.Error()
		return rr
	}

	req := Request{
		Protocol: Protocol,
		Kind:     KindChecks,
		Checks:   checks,
	}
	payload, err := json.Marshal(req)
	if err != nil {
		rr.Error = "marshal request: " + err.Error()
		return rr
	}

	timeout := DefaultTimeout
	if spec.Timeout != "" {
		if d, terr := time.ParseDuration(spec.Timeout); terr == nil {
			if d > 0 && d < DefaultExecTimeout {
				timeout = d
			}
		}
	}
	if timeout > DefaultExecTimeout {
		timeout = DefaultExecTimeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if workdir != "" {
		cmd.Dir = workdir
	}

	err = cmd.Run()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded || (errors.Is(err, context.DeadlineExceeded)) {
			rr.Error = fmt.Sprintf("plugin %q timed out after %s", spec.Name, timeout)
		} else {
			rr.Error = fmt.Sprintf("plugin %q: %s; stderr=%s", spec.Name, err.Error(), strings.TrimSpace(stderr.String()))
		}
		return rr
	}

	var resp Response
	if err := json.Unmarshal(stdout.Bytes(), &resp); err != nil {
		rr.Error = fmt.Sprintf("plugin %q: bad JSON: %v; raw=%s", spec.Name, err, truncate(stdout.String(), 256))
		return rr
	}
	if resp.Protocol != Protocol {
		rr.Error = fmt.Sprintf("plugin %q: protocol mismatch: got %q, want %q", spec.Name, resp.Protocol, Protocol)
		return rr
	}
	if resp.Error != "" {
		rr.Error = fmt.Sprintf("plugin %q reported: %s", spec.Name, resp.Error)
		return rr
	}
	rr.Results = resp.Results
	return rr
}

// resolveCommand turns a spec into argv. Spec.Command (when set) is split
// with simple shell-style rules; Spec.Binary is looked up on PATH.
func resolveCommand(spec PluginSpec) ([]string, error) {
	if spec.Command != "" {
		parts := strings.Fields(spec.Command)
		if len(parts) == 0 {
			return nil, fmt.Errorf("plugin %q has empty command", spec.Name)
		}
		return append(parts, spec.Args...), nil
	}
	if spec.Binary != "" {
		if path, err := loadLookPath(spec.Binary); err == nil {
			return append([]string{path}, spec.Args...), nil
		} else if strings.ContainsRune(spec.Binary, os.PathSeparator) || strings.HasPrefix(spec.Binary, ".") {
			// Explicit relative or absolute path; exec.Command will resolve.
			return append([]string{spec.Binary}, spec.Args...), nil
		}
		return nil, fmt.Errorf("plugin %q: binary %q not on PATH", spec.Name, spec.Binary)
	}
	return nil, fmt.Errorf("plugin %q needs either binary or command", spec.Name)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// EncodeResponse is a small helper used by companion plugin binaries — it
// marshals Response with the protocol field filled, ready to write to
// stdout. Not used by the host (the host reads, not writes).
func EncodeResponse(results []Result) ([]byte, error) {
	return json.Marshal(Response{Protocol: Protocol, Results: results})
}

// DecodeRequest parses stdin bytes into a Request with the protocol field
// validated. Companion plugin binaries use this on startup.
func DecodeRequest(r io.Reader) (Request, error) {
	var req Request
	if err := json.NewDecoder(r).Decode(&req); err != nil {
		return req, err
	}
	if req.Protocol != Protocol {
		return req, fmt.Errorf("protocol mismatch: got %q, want %q", req.Protocol, Protocol)
	}
	if req.Kind != KindChecks {
		return req, fmt.Errorf("unknown kind %q", req.Kind)
	}
	return req, nil
}
