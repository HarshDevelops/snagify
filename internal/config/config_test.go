package config

import (
	"os"
	"path/filepath"
	"testing"
)

const sample = `
version: 1
project:
  name: checkout-service
runtimes:
  node: ">=20 <23"
  java: "17"
  maven: ">=3.8"
  docker: "required"
env:
  example_file: ".env.example"
  actual_file: ".env"
  required:
    - DATABASE_URL
    - JWT_SECRET
    - REDIS_URL
ports:
  must_be_free:
    - 8080
  must_be_listening:
    - 5432
    - 6379
git:
  require_clean: false
  warn_if_wrong_branch: false
ignore:
  runtimes:
    - rust
    - cargo
`

func TestLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte(sample), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Project.Name != "checkout-service" {
		t.Errorf("name = %q", cfg.Project.Name)
	}
	if cfg.Runtimes["node"] != ">=20 <23" {
		t.Errorf("node req = %q", cfg.Runtimes["node"])
	}
	if cfg.Runtimes["docker"] != "required" {
		t.Errorf("docker req = %q", cfg.Runtimes["docker"])
	}
	if len(cfg.Env.Required) != 3 {
		t.Errorf("required keys = %v", cfg.Env.Required)
	}
	if len(cfg.Ports.MustBeListening) != 2 || cfg.Ports.MustBeListening[0] != 5432 {
		t.Errorf("listening ports = %v", cfg.Ports.MustBeListening)
	}
	if len(cfg.Ports.MustBeFree) != 1 || cfg.Ports.MustBeFree[0] != 8080 {
		t.Errorf("free ports = %v", cfg.Ports.MustBeFree)
	}
	if !cfg.IgnoredRuntime("Rust") || !cfg.IgnoredRuntime("cargo") {
		t.Error("expected rust and cargo to be ignored")
	}
	if cfg.IgnoredRuntime("node") {
		t.Error("node should not be ignored")
	}
}

func TestLoadMalformed(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, FileName)
	if err := os.WriteFile(path, []byte("version: 1\n  bad: : :\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestLoadMissing(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Error("expected error for missing file")
	}
}
