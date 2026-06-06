package fixplan

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ApplyOptions controls safe-fix execution.
type ApplyOptions struct {
	DryRun      bool
	Yes         bool
	Force       bool
	Stdout      io.Writer
	ExamplePath string
	ActualPath  string
	ProjectRoot string
}

// ApplyResult records what happened.
type ApplyResult struct {
	Applied []string
	Skipped []string
	Errors  []string
}

// ApplySafe executes only the safe fixes in the plan. It never installs
// runtimes, never modifies PATH/shell files, never writes real secret values,
// and never starts Docker services.
func ApplySafe(plan FixPlan, opts ApplyOptions) ApplyResult {
	var result ApplyResult
	w := opts.Stdout
	if w == nil {
		w = io.Discard
	}

	for _, action := range plan.Safe {
		switch action.ID {
		case "create-env-from-example":
			result.merge(applyCreateEnvFromExample(opts, w))
		case "append-missing-env-keys":
			result.merge(applyAppendMissingKeys(action, opts, w))
		default:
			if strings.HasPrefix(action.ID, "docker-pull-") {
				result.merge(applyDockerPull(action, opts, w))
			} else {
				result.Skipped = append(result.Skipped, action.Title)
			}
		}
	}

	// snagify init equivalent: create .snagify.yaml if missing.
	yamlPath := filepath.Join(opts.ProjectRoot, ".snagify.yaml")
	if _, err := os.Stat(yamlPath); os.IsNotExist(err) {
		result.merge(applyInitConfig(opts, w, yamlPath))
	}

	return result
}

func applyCreateEnvFromExample(opts ApplyOptions, w io.Writer) ApplyResult {
	var r ApplyResult
	if opts.DryRun {
		fmt.Fprintln(w, "[dry-run] Would create .env from .env.example (blank secret values)")
		r.Applied = append(r.Applied, "create-env-from-example (dry-run)")
		return r
	}
	// Read keys+defaults from example; never invent secret values.
	content, err := envFromExample(opts.ExamplePath)
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("read %s: %v", opts.ExamplePath, err))
		return r
	}
	if err := os.WriteFile(opts.ActualPath, []byte(content), 0o600); err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("write %s: %v", opts.ActualPath, err))
		return r
	}
	fmt.Fprintf(w, "Created %s from %s\n", opts.ActualPath, opts.ExamplePath)
	r.Applied = append(r.Applied, "create-env-from-example")
	return r
}

func applyAppendMissingKeys(action FixAction, opts ApplyOptions, w io.Writer) ApplyResult {
	var r ApplyResult
	if opts.DryRun {
		fmt.Fprintf(w, "[dry-run] Would append to .env: %s\n", strings.Join(action.AppliesTo, ", "))
		r.Applied = append(r.Applied, "append-missing-env-keys (dry-run)")
		return r
	}
	f, err := os.OpenFile(opts.ActualPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		r.Errors = append(r.Errors, fmt.Sprintf("open %s: %v", opts.ActualPath, err))
		return r
	}
	defer f.Close()

	for _, k := range action.AppliesTo {
		if _, err := fmt.Fprintf(f, "%s=\n", k); err != nil {
			r.Errors = append(r.Errors, fmt.Sprintf("write key %s: %v", k, err))
			continue
		}
		fmt.Fprintf(w, "Appended %s= to .env\n", k)
	}
	r.Applied = append(r.Applied, "append-missing-env-keys")
	return r
}

func applyDockerPull(action FixAction, opts ApplyOptions, w io.Writer) ApplyResult {
	var r ApplyResult
	if opts.DryRun {
		fmt.Fprintf(w, "[dry-run] Would run: %s\n", strings.Join(action.Commands, "; "))
		r.Applied = append(r.Applied, action.ID+" (dry-run)")
		return r
	}
	if !opts.Yes {
		r.Skipped = append(r.Skipped, action.ID+" (needs --yes)")
		return r
	}
	// Execute docker pull.
	for _, cmd := range action.Commands {
		if strings.HasPrefix(cmd, "docker pull ") {
			if err := execDockerPull(strings.TrimPrefix(cmd, "docker pull "), w); err != nil {
				r.Errors = append(r.Errors, err.Error())
				return r
			}
			r.Applied = append(r.Applied, action.ID)
		}
	}
	return r
}

func applyInitConfig(opts ApplyOptions, w io.Writer, yamlPath string) ApplyResult {
	var r ApplyResult
	if opts.DryRun {
		fmt.Fprintln(w, "[dry-run] Would run `snagify init` to create .snagify.yaml")
		r.Applied = append(r.Applied, "init-config (dry-run)")
	}
	// Not implemented here — caller (CLI) invokes init logic directly.
	return r
}

// envFromExample reads .env.example and returns a .env file content where:
// - comments are preserved
// - keys with non-sensitive defaults keep their defaults
// - blank values remain blank
// - actual secret values are NEVER included
func envFromExample(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	var b strings.Builder
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// Preserve comments and blank lines.
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			b.WriteString(line + "\n")
			continue
		}
		// For export directives.
		bare := strings.TrimPrefix(trimmed, "export ")
		eq := strings.Index(bare, "=")
		if eq < 0 {
			b.WriteString(line + "\n")
			continue
		}
		key := bare[:eq]
		val := bare[eq+1:]
		// Only keep default values that are NOT secrets.
		// For blank values, keep blank; for non-blank non-sensitive values,
		// keep the default; for blank sensitive keys, keep blank.
		b.WriteString(key + "=" + safeDefault(key, val) + "\n")
	}
	return b.String(), nil
}

// safeDefault returns the default value from .env.example when it is safe to
// copy (non-sensitive), otherwise returns blank.
func safeDefault(key, val string) string {
	if val == "" {
		return ""
	}
	upper := strings.ToUpper(key)
	sensitiveTerms := []string{
		"SECRET", "PASSWORD", "API_KEY", "TOKEN", "AUTH", "CREDENTIAL",
		"PRIVATE", "SIGNING",
	}
	for _, t := range sensitiveTerms {
		if strings.Contains(upper, t) {
			return "" // blank out sensitive keys even if example has a placeholder
		}
	}
	return val
}

// execDockerPull runs "docker pull <image>" and streams output to w.
func execDockerPull(image string, w io.Writer) error {
	// Use os/exec to run docker pull. Import locally to avoid a top-level import
	// that would pull in unnecessary dependencies.
	return runCmd(w, "docker", "pull", image)
}

func (r *ApplyResult) merge(other ApplyResult) {
	r.Applied = append(r.Applied, other.Applied...)
	r.Skipped = append(r.Skipped, other.Skipped...)
	r.Errors = append(r.Errors, other.Errors...)
}
