package initcfg

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/harshdevelops/snagify/internal/config"
	"github.com/harshdevelops/snagify/internal/model"
	"gopkg.in/yaml.v3"
)

func TestGenerate_ProducesParseableConservativeConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "docker-compose.yml"), []byte("services: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte("DATABASE_URL=\nJWT_SECRET=\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	snap := model.Snapshot{
		Project: model.ProjectInfo{
			Name:      "checkout-service",
			Root:      dir,
			Manifests: []string{"pom.xml"},
		},
		Runtimes: model.Runtimes{
			Java:  model.VersionInfo{Version: "17.0.9", Present: true},
			Maven: model.VersionInfo{Version: "3.9.6", Present: true},
		},
	}

	doc := Generate(snap, dir)

	// Must parse back into a valid config.
	var cfg config.Config
	if err := yaml.Unmarshal([]byte(doc), &cfg); err != nil {
		t.Fatalf("generated config does not parse: %v\n%s", err, doc)
	}

	if cfg.Project.Name != "checkout-service" {
		t.Errorf("name = %q", cfg.Project.Name)
	}
	if got := cfg.Runtimes["java"]; got != ">=17" {
		t.Errorf("java req = %q, want >=17", got)
	}
	if got := cfg.Runtimes["maven"]; got != ">=3" {
		t.Errorf("maven req = %q, want >=3", got)
	}
	if cfg.Runtimes["docker"] != "required" {
		t.Errorf("docker req = %q, want required", cfg.Runtimes["docker"])
	}
	if cfg.Env.ExampleFile != ".env.example" {
		t.Errorf("example_file = %q", cfg.Env.ExampleFile)
	}
	// DATABASE_URL and JWT_SECRET are sensitive blanks => required.
	if len(cfg.Env.Required) != 2 {
		t.Errorf("required keys = %v (want 2 sensitive keys)", cfg.Env.Required)
	}
	// No keys should end up as optional for this simple fixture.
	if len(cfg.Env.Optional) != 0 {
		t.Errorf("expected no optional keys, got %v", cfg.Env.Optional)
	}

	// Must never include absolute paths.
	if strings.Contains(doc, dir) {
		t.Errorf("generated config leaked absolute path:\n%s", doc)
	}
}

func TestGenerate_SpecFixtureClassifiesCorrectly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env.example"), []byte(fixtureEnvExample), 0o644); err != nil {
		t.Fatal(err)
	}

	snap := model.Snapshot{Project: model.ProjectInfo{Name: "demo", Root: dir}}
	doc := Generate(snap, dir)

	var cfg config.Config
	if err := yaml.Unmarshal([]byte(doc), &cfg); err != nil {
		t.Fatalf("generated config does not parse: %v\n%s", err, doc)
	}

	// Required list must be small (the 4 sensitive keys).
	if len(cfg.Env.Required) > 4 {
		t.Errorf("required list too large: %v", cfg.Env.Required)
	}
	for _, k := range cfg.Env.Required {
		if strings.HasPrefix(k, "FALLBACK_") {
			t.Errorf("disabled fallback key %s must not be required", k)
		}
	}

	// Fallback keys must end up optional.
	optSet := map[string]bool{}
	for _, k := range cfg.Env.Optional {
		optSet[k] = true
	}
	for _, k := range []string{"FALLBACK_1_API_KEY", "FALLBACK_2_API_KEY"} {
		if !optSet[k] {
			t.Errorf("expected %s to be in optional list, got optional=%v", k, cfg.Env.Optional)
		}
	}

	// Tuning keys with defaults must be in recommended, not required.
	for _, k := range cfg.Env.Required {
		for _, prefix := range []string{"MAX_", "RETRY_", "TIMEOUT_"} {
			if strings.HasPrefix(k, prefix) {
				t.Errorf("tuning key %s should not be required", k)
			}
		}
	}
}
