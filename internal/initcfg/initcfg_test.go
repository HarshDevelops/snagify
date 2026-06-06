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
	if len(cfg.Env.Required) != 2 {
		t.Errorf("required keys = %v", cfg.Env.Required)
	}

	// Must never include absolute paths.
	if strings.Contains(doc, dir) {
		t.Errorf("generated config leaked absolute path:\n%s", doc)
	}
}
