// Package fixplan turns a check report into a structured fix plan with safe,
// guided, and unfixable actions. It never executes anything — callers are
// responsible for acting on or printing the plan.
package fixplan

import (
	"fmt"
	"os"
	"strings"

	"github.com/harshdevelops/snagify/internal/report"
)

// FixPlan groups fix actions by applicability.
type FixPlan struct {
	Project   string
	Safe      []FixAction
	Guided    []FixAction
	Manual    []FixAction
	Unfixable []FixAction
}

// FixAction describes a single fix.
type FixAction struct {
	ID                   string
	Title                string
	Description          string
	Commands             []string
	Safe                 bool
	RequiresConfirmation bool
	AppliesTo            []string
}

// HasFixes reports whether the plan contains any safe or guided fixes.
func (p FixPlan) HasFixes() bool {
	return len(p.Safe) > 0 || len(p.Guided) > 0
}

// Build produces a FixPlan from a report, an optional .env.example path, and
// the project root. It never reads actual .env values.
func Build(r report.Report, projectRoot, envExamplePath, envActualPath string) FixPlan {
	plan := FixPlan{Project: r.Project}

	for _, it := range r.Items {
		if it.Severity != report.Critical && it.Severity != report.Warning {
			continue
		}
		addActions(&plan, it, projectRoot, envExamplePath, envActualPath)
	}

	return plan
}

func addActions(plan *FixPlan, it report.Item, projectRoot, examplePath, actualPath string) {
	switch it.Category {
	case "Env":
		addEnvFixes(plan, it, examplePath, actualPath)
	case "Runtime":
		addRuntimeFixes(plan, it)
	case "Service", "Port":
		addServiceFixes(plan, it)
	case "Docker":
		addDockerFixes(plan, it, projectRoot)
	default:
		plan.Unfixable = append(plan.Unfixable, FixAction{
			ID:          "info-" + strings.ToLower(it.Name),
			Title:       it.Name + " drift",
			Description: fmt.Sprintf("%s: %s (found %s, expected %s)", it.Category, it.Name, it.Found, it.Expected),
			Safe:        false,
			AppliesTo:   []string{it.Name},
		})
	}
}

func addEnvFixes(plan *FixPlan, it report.Item, examplePath, actualPath string) {
	envExists := fileExists(actualPath)

	if !envExists && examplePath != "" && fileExists(examplePath) {
		plan.Safe = append(plan.Safe, FixAction{
			ID:                   "create-env-from-example",
			Title:                "Create .env from .env.example",
			Description:          "Copies .env.example to .env, preserving non-secret defaults. Secret fields are left blank.",
			Safe:                 true,
			RequiresConfirmation: false,
			AppliesTo:            []string{".env"},
		})
	}

	// Extract missing key names from blocker text.
	missing := extractMissingKeys(it.Blocker)
	if len(missing) > 0 {
		plan.Safe = append(plan.Safe, FixAction{
			ID:    "append-missing-env-keys",
			Title: "Append missing keys to .env as blank placeholders",
			Description: fmt.Sprintf(
				"Appends %s to .env with blank values so the file is valid. You still need to set real values.",
				strings.Join(missing, ", ")),
			Safe:                 true,
			RequiresConfirmation: false,
			AppliesTo:            missing,
		})
	}
}

func addRuntimeFixes(plan *FixPlan, it report.Item) {
	rt := strings.ToLower(it.Name)
	if strings.Contains(it.Found, "missing") || it.Found == "missing" {
		plan.Guided = append(plan.Guided, installGuide(rt, ""))
		return
	}
	// Version mismatch — suggest version manager commands.
	expected := it.Expected
	plan.Guided = append(plan.Guided, versionGuide(rt, it.Found, expected))
}

func addServiceFixes(plan *FixPlan, it report.Item) {
	plan.Guided = append(plan.Guided, FixAction{
		ID:          "service-" + strings.ToLower(it.Name),
		Title:       it.Name + " is not running",
		Description: fmt.Sprintf("Start the service so %s is reachable.", it.Name),
		Commands:    []string{"docker compose up -d " + strings.ToLower(it.Name)},
		Safe:        false,
		AppliesTo:   []string{it.Name},
	})
}

func addDockerFixes(plan *FixPlan, it report.Item, projectRoot string) {
	switch {
	case strings.Contains(it.Name, "runtime") || strings.Contains(it.Found, "missing"):
		plan.Guided = append(plan.Guided, installGuide("docker", ""))
	case strings.Contains(it.Name, ".yml") || strings.Contains(it.Name, ".yaml"):
		plan.Unfixable = append(plan.Unfixable, FixAction{
			ID:          "compose-file-" + it.Name,
			Title:       "Missing compose file: " + it.Name,
			Description: fmt.Sprintf("Compose file %q is missing. Create it or commit it to the repo.", it.Name),
			AppliesTo:   []string{it.Name},
		})
	default:
		// Required image not pulled.
		plan.Safe = append(plan.Safe, FixAction{
			ID:                   "docker-pull-" + strings.ToLower(it.Name),
			Title:                "Pull required Docker image: " + it.Name,
			Description:          "Pulls the image locally so it is available.",
			Commands:             []string{"docker pull " + it.Name},
			Safe:                 true,
			RequiresConfirmation: true,
			AppliesTo:            []string{it.Name},
		})
	}
}

func installGuide(rt, version string) FixAction {
	var cmds []string
	rt = strings.TrimSpace(rt)
	switch rt {
	case "node":
		cmds = []string{"nvm install " + coalesce(version, "20"), "nvm use " + coalesce(version, "20")}
	case "java":
		v := coalesce(version, "17.0.10-tem")
		cmds = []string{"sdk install java " + v, "sdk use java " + v}
	case "maven":
		cmds = []string{"brew install maven", "sdk install maven"}
	case "python":
		cmds = []string{"pyenv install " + coalesce(version, "3.11"), "pyenv local " + coalesce(version, "3.11")}
	case "docker":
		cmds = []string{
			"macOS: Install Docker Desktop from https://www.docker.com/products/docker-desktop/",
			"Linux: curl -fsSL https://get.docker.com | sh",
		}
	default:
		cmds = []string{"# Install " + rt + " using your system package manager"}
	}
	return FixAction{
		ID:          "install-" + rt,
		Title:       "Install " + rt,
		Description: fmt.Sprintf("%s is missing. Suggested install commands:", rt),
		Commands:    cmds,
		Safe:        false,
		AppliesTo:   []string{rt},
	}
}

func versionGuide(rt, found, expected string) FixAction {
	var cmds []string
	switch rt {
	case "node":
		cmds = []string{
			"nvm install " + expected,
			"nvm use " + expected,
			"# If .nvmrc exists: nvm use",
		}
	case "java":
		cmds = []string{
			"sdk install java " + expected,
			"sdk use java " + expected,
		}
	case "python":
		cmds = []string{
			"pyenv install " + expected,
			"pyenv local " + expected,
		}
	default:
		cmds = []string{"# Switch " + rt + " to version " + expected + " using your version manager"}
	}
	return FixAction{
		ID:          "version-" + rt,
		Title:       fmt.Sprintf("%s: switch to %s (found %s)", rt, expected, found),
		Description: fmt.Sprintf("Suggested commands to switch %s from %s to %s:", rt, found, expected),
		Commands:    cmds,
		Safe:        false,
		AppliesTo:   []string{rt},
	}
}

// extractMissingKeys parses a blocker string like:
// "Required env keys are absent: KEY1, KEY2"
func extractMissingKeys(blocker string) []string {
	const needle = ": "
	i := strings.LastIndex(blocker, needle)
	if i < 0 {
		return nil
	}
	raw := strings.TrimSpace(blocker[i+len(needle):])
	var keys []string
	for _, k := range strings.Split(raw, ",") {
		k = strings.TrimSpace(k)
		if k != "" {
			keys = append(keys, k)
		}
	}
	return keys
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}

func coalesce(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
