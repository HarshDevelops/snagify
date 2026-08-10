// Package config loads and represents the .snagify.yaml repo requirement file.
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// FileName is the conventional config filename in a project root.
const FileName = ".snagify.yaml"

// Config is the parsed .snagify.yaml requirement set.
type Config struct {
	Version  int               `yaml:"version"`
	Project  Project           `yaml:"project"`
	Runtimes map[string]string `yaml:"runtimes"`
	Env      Env               `yaml:"env"`
	Ports    Ports             `yaml:"ports"`
	Git      Git               `yaml:"git"`
	Ignore   Ignore            `yaml:"ignore"`

	// v0.3 optional sections.
	Path     Path     `yaml:"path"`
	Docker   Docker   `yaml:"docker"`
	Services Services `yaml:"services"`
	Network  Network  `yaml:"network"`
	TLS      TLS      `yaml:"tls"`
	System   System   `yaml:"system"`
	Databases Databases `yaml:"databases"`

	// v0.5 optional sections.
	Checks  Checks  `yaml:"checks"`
	Plugins Plugins `yaml:"plugins"`
}

// Checks groups user-defined shell checks under one section so additional
// check families (e.g. http probes, regex asserts) can be added later
// without colliding with custom-check numbering.
type Checks struct {
	Custom []CustomCheck `yaml:"custom"`
}

// CustomCheck is one entry of `checks.custom` in .snagify.yaml. Snagify runs
// the command, captures exit code + truncated stdout/stderr, and converts
// the result into a report item. Severity defaults to "warning" if unset.
type CustomCheck struct {
	Name     string   `yaml:"name"`
	Run      []string `yaml:"run"`
	Severity string   `yaml:"severity"`
	Message  string   `yaml:"message"`
	Timeout  string   `yaml:"timeout"`
	Workdir  string   `yaml:"workdir"`
}

// Plugins describes external snagify-plugin-* binaries. Each is a
// subprocess that speaks the snagify/plugin/v1 JSON contract.
type Plugins struct {
	Items []PluginSpec `yaml:"items"`
}

// PluginSpec is one plugin invocation. Either Binary (on $PATH) or Command
// (relative to the project root, may include args) must be set.
type PluginSpec struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description,omitempty"`
	Binary      string `yaml:"binary,omitempty"`
	Command     string `yaml:"command,omitempty"`
	Args        []string `yaml:"args,omitempty"`
	Timeout     string `yaml:"timeout,omitempty"`
}

// Path describes PATH/executable requirements.
type Path struct {
	RequireCommands      []string `yaml:"require_commands"`
	WarnIfDifferentPaths []string `yaml:"warn_if_different_paths"`
}

// Docker describes container/compose requirements.
type Docker struct {
	Required           bool     `yaml:"required"`
	ComposeFiles       []string `yaml:"compose_files"`
	RequiredImages     []string `yaml:"required_images"`
	RequiredContainers []string `yaml:"required_containers"`
}

// Services describes active TCP reachability probes.
type Services struct {
	TCP []TCPService `yaml:"tcp"`
}

// TCPService is a single TCP reachability target.
type TCPService struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Required  bool   `yaml:"required"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// Network describes DNS/HTTP/proxy probes.
type Network struct {
	DNS   []string    `yaml:"dns"`
	HTTP  []HTTPProbe `yaml:"http"`
	Proxy ProxyConfig `yaml:"proxy"`
}

// HTTPProbe is a single HTTP endpoint probe.
type HTTPProbe struct {
	Name           string `yaml:"name"`
	URL            string `yaml:"url"`
	Method         string `yaml:"method"`
	ExpectedStatus int    `yaml:"expected_status"`
	Required       bool   `yaml:"required"`
}

// ProxyConfig controls proxy env capture/comparison.
type ProxyConfig struct {
	CaptureEnv         bool `yaml:"capture_env"`
	WarnIfProxyDiffers bool `yaml:"warn_if_proxy_differs"`
}

// TLS describes TLS endpoint probes.
type TLS struct {
	Endpoints []TLSEndpoint `yaml:"endpoints"`
}

// TLSEndpoint is a single TLS probe target.
type TLSEndpoint struct {
	Name                    string `yaml:"name"`
	Host                    string `yaml:"host"`
	Port                    int    `yaml:"port"`
	ServerName              string `yaml:"server_name"`
	Required                bool   `yaml:"required"`
	WarnIfExpiresWithinDays int    `yaml:"warn_if_expires_within_days"`
}

// System describes expected locale/timezone/filesystem/arch.
type System struct {
	Timezone        string   `yaml:"timezone"`
	Locale          string   `yaml:"locale"`
	CaseSensitiveFS *bool    `yaml:"case_sensitive_fs"`
	AllowedArch     []string `yaml:"allowed_arch"`
}

// Project holds project-level metadata.
type Project struct {
	Name string `yaml:"name"`
}

// Env describes env-file requirements.
// Required keys failing => critical. Recommended keys missing => warning.
// Optional keys missing => ignored (silent).
type Env struct {
	ExampleFile string   `yaml:"example_file"`
	ActualFile  string   `yaml:"actual_file"`
	Required    []string `yaml:"required"`
	Recommended []string `yaml:"recommended"`
	Optional    []string `yaml:"optional"`
}

// Ports describes port requirements.
type Ports struct {
	MustBeFree      []int `yaml:"must_be_free"`
	MustBeListening []int `yaml:"must_be_listening"`
}

// Git describes git-related checks.
type Git struct {
	RequireClean      bool   `yaml:"require_clean"`
	WarnIfWrongBranch bool   `yaml:"warn_if_wrong_branch"`
	RequireBranch     string `yaml:"require_branch"`
	WarnIfDirty       bool   `yaml:"warn_if_dirty"`
}

// Databases describes protocol-level database probes. These probes send no
// credentials: Redis uses an unauthenticated PING, Postgres uses an
// SSLRequest packet (no StartupMessage, no auth), and MySQL reads only the
// server's Initial Handshake packet. They confirm that the server speaks
// the expected wire protocol on the target port.
type Databases struct {
	Postgres []PostgresService `yaml:"postgres"`
	MySQL    []MySQLService    `yaml:"mysql"`
	Redis    []RedisService    `yaml:"redis"`
}

// PostgresService is a single Postgres wire-protocol probe target.
type PostgresService struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Required  bool   `yaml:"required"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// MySQLService is a single MySQL wire-protocol probe target.
type MySQLService struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Required  bool   `yaml:"required"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// RedisService is a single Redis wire-protocol probe target.
type RedisService struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	Required  bool   `yaml:"required"`
	TimeoutMS int    `yaml:"timeout_ms"`
}

// HasActiveProbes reports whether the config declares any probe that requires
// network/service/TLS activity.
func (c Config) HasActiveProbes() bool {
	return len(c.Services.TCP) > 0 ||
		len(c.Network.DNS) > 0 ||
		len(c.Network.HTTP) > 0 ||
		len(c.TLS.Endpoints) > 0 ||
		len(c.Databases.Postgres) > 0 ||
		len(c.Databases.MySQL) > 0 ||
		len(c.Databases.Redis) > 0
}

// Ignore lists items to skip during checks.
type Ignore struct {
	Runtimes []string `yaml:"runtimes"`
}

// Load reads and parses a config file. It returns a clear error for missing
// or malformed files so the CLI can exit with code 2.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: not valid YAML (%w)", path, err)
	}
	if c.Version == 0 {
		c.Version = 1
	}
	return c, nil
}

// IgnoredRuntime reports whether a runtime (case-insensitive) is in the ignore
// list.
func (c Config) IgnoredRuntime(name string) bool {
	for _, r := range c.Ignore.Runtimes {
		if equalFold(r, name) {
			return true
		}
	}
	return false
}

func equalFold(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca, cb := a[i], b[i]
		if 'A' <= ca && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if 'A' <= cb && cb <= 'Z' {
			cb += 'a' - 'A'
		}
		if ca != cb {
			return false
		}
	}
	return true
}
