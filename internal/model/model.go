// Package model defines the data structures captured in a Snagify snapshot.
package model

// Snapshot is the full capture of a machine/project state at a point in time.
type Snapshot struct {
	Timestamp   string      `json:"timestamp"`
	Hostname    string      `json:"hostname"`
	Project     ProjectInfo `json:"project"`
	Environment Environment `json:"environment"`
	Runtimes    Runtimes    `json:"runtimes"`
	Services    Services    `json:"services"`
	EnvFiles    EnvFiles    `json:"env_files"`
}

// ProjectInfo describes the detected project root and its manifests.
type ProjectInfo struct {
	Root      string   `json:"root"`
	Manifests []string `json:"manifests"` // e.g. package.json, go.mod
	Name      string   `json:"name,omitempty"`
}

// Environment captures OS-level details.
type Environment struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	OSVersion  string `json:"os_version,omitempty"`
	GitVersion string `json:"git_version"`
}

// Runtimes holds detected language/tool runtimes.
type Runtimes struct {
	Node   VersionInfo `json:"node"`
	NPM    VersionInfo `json:"npm"`
	PNPM   VersionInfo `json:"pnpm"`
	Yarn   VersionInfo `json:"yarn"`
	Python VersionInfo `json:"python"`
	Pip    VersionInfo `json:"pip"`
	UV     VersionInfo `json:"uv"`
	Java   VersionInfo `json:"java"`
	Maven  VersionInfo `json:"maven"`
	Gradle VersionInfo `json:"gradle"`
	Go     VersionInfo `json:"go"`
	Rust   VersionInfo `json:"rust"`
	Cargo  VersionInfo `json:"cargo"`
	Docker VersionInfo `json:"docker"`
}

// VersionInfo records whether a tool is present and its version.
type VersionInfo struct {
	Version string `json:"version"`
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
}

// Services holds network service state (currently ports).
type Services struct {
	Ports map[int]PortInfo `json:"ports"`
}

// PortInfo records whether a common dev port is listening.
type PortInfo struct {
	Port      int    `json:"port"`
	Service   string `json:"service,omitempty"`
	Listening bool   `json:"listening"`
}

// EnvFiles records presence of env files and missing keys.
type EnvFiles struct {
	EnvExampleExists bool     `json:"env_example_exists"`
	EnvExists        bool     `json:"env_exists"`
	MissingKeys      []string `json:"missing_keys"`
}

// RuntimeField pairs a display name with an accessor into Runtimes,
// enabling generic iteration over all runtimes.
type RuntimeField struct {
	Name string
	Get  func(r Runtimes) VersionInfo
}

// RuntimeFields enumerates every runtime in a stable display order.
var RuntimeFields = []RuntimeField{
	{"Node", func(r Runtimes) VersionInfo { return r.Node }},
	{"npm", func(r Runtimes) VersionInfo { return r.NPM }},
	{"pnpm", func(r Runtimes) VersionInfo { return r.PNPM }},
	{"Yarn", func(r Runtimes) VersionInfo { return r.Yarn }},
	{"Python", func(r Runtimes) VersionInfo { return r.Python }},
	{"pip", func(r Runtimes) VersionInfo { return r.Pip }},
	{"uv", func(r Runtimes) VersionInfo { return r.UV }},
	{"Java", func(r Runtimes) VersionInfo { return r.Java }},
	{"Maven", func(r Runtimes) VersionInfo { return r.Maven }},
	{"Gradle", func(r Runtimes) VersionInfo { return r.Gradle }},
	{"Go", func(r Runtimes) VersionInfo { return r.Go }},
	{"Rust", func(r Runtimes) VersionInfo { return r.Rust }},
	{"Cargo", func(r Runtimes) VersionInfo { return r.Cargo }},
	{"Docker", func(r Runtimes) VersionInfo { return r.Docker }},
}
