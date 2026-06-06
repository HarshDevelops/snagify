package config

import (
	"os"
	"path/filepath"
	"testing"
)

// v02Config is a minimal v0.2-era config with no v0.3 sections.
const v02Config = `
version: 1
project:
  name: legacy-app
runtimes:
  node: ">=18"
env:
  required:
    - DATABASE_URL
ports:
  must_be_listening:
    - 5432
`

const v03Config = `
version: 1
project:
  name: checkout-service
git:
  require_branch: main
  require_clean: false
  warn_if_dirty: true
runtimes:
  java: "17"
path:
  require_commands:
    - node
    - java
  warn_if_different_paths:
    - node
docker:
  required: true
  compose_files:
    - docker-compose.yml
  required_images:
    - postgres:15
  required_containers:
    - postgres
services:
  tcp:
    - name: postgres
      host: localhost
      port: 5432
      required: true
network:
  dns:
    - registry.npmjs.org
  http:
    - name: npm-registry
      url: https://registry.npmjs.org
      method: HEAD
      expected_status: 200
      required: true
  proxy:
    capture_env: true
    warn_if_proxy_differs: true
tls:
  endpoints:
    - name: npm-registry
      host: registry.npmjs.org
      port: 443
      server_name: registry.npmjs.org
      required: true
      warn_if_expires_within_days: 14
system:
  timezone: Asia/Kolkata
  locale: en_US.UTF-8
  case_sensitive_fs: false
  allowed_arch:
    - arm64
    - amd64
ignore:
  runtimes:
    - rust
`

func loadStr(t *testing.T, s string) Config {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, FileName)
	if err := os.WriteFile(p, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(p)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return cfg
}

func TestLoad_V02StillParses(t *testing.T) {
	cfg := loadStr(t, v02Config)
	if cfg.Project.Name != "legacy-app" {
		t.Errorf("name = %q", cfg.Project.Name)
	}
	// v0.3 sections must be zero-valued and not cause errors.
	if cfg.HasActiveProbes() {
		t.Error("legacy config should declare no active probes")
	}
	if cfg.Docker.Required {
		t.Error("legacy config should not require docker")
	}
}

func TestLoad_V03FullParses(t *testing.T) {
	cfg := loadStr(t, v03Config)

	if cfg.Git.RequireBranch != "main" || !cfg.Git.WarnIfDirty {
		t.Errorf("git = %+v", cfg.Git)
	}
	if len(cfg.Path.RequireCommands) != 2 {
		t.Errorf("require_commands = %v", cfg.Path.RequireCommands)
	}
	if !cfg.Docker.Required || len(cfg.Docker.RequiredImages) != 1 {
		t.Errorf("docker = %+v", cfg.Docker)
	}
	if len(cfg.Services.TCP) != 1 || cfg.Services.TCP[0].Port != 5432 {
		t.Errorf("services = %+v", cfg.Services)
	}
	if len(cfg.Network.HTTP) != 1 || cfg.Network.HTTP[0].ExpectedStatus != 200 {
		t.Errorf("network = %+v", cfg.Network)
	}
	if !cfg.Network.Proxy.CaptureEnv {
		t.Error("expected proxy capture_env true")
	}
	if len(cfg.TLS.Endpoints) != 1 || cfg.TLS.Endpoints[0].WarnIfExpiresWithinDays != 14 {
		t.Errorf("tls = %+v", cfg.TLS)
	}
	if cfg.System.Timezone != "Asia/Kolkata" {
		t.Errorf("system.timezone = %q", cfg.System.Timezone)
	}
	if cfg.System.CaseSensitiveFS == nil || *cfg.System.CaseSensitiveFS {
		t.Error("expected case_sensitive_fs false (non-nil)")
	}
	if len(cfg.System.AllowedArch) != 2 {
		t.Errorf("allowed_arch = %v", cfg.System.AllowedArch)
	}
	if !cfg.HasActiveProbes() {
		t.Error("expected active probes")
	}
}
