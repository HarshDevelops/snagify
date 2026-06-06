package model

// GitInfo captures git repository state. Present is false when the project is
// not inside a git work tree.
type GitInfo struct {
	Present   bool   `json:"present"`
	Branch    string `json:"branch,omitempty"`
	Commit    string `json:"commit,omitempty"`
	Dirty     bool   `json:"dirty"`
	Untracked bool   `json:"untracked"`
	Ahead     int    `json:"ahead,omitempty"`
	Behind    int    `json:"behind,omitempty"`
	Error     string `json:"error,omitempty"`
}

// PathInfo captures PATH entries and resolved executable locations. Paths are
// redacted so the user's home directory becomes "~".
type PathInfo struct {
	PathEntries []string            `json:"path_entries,omitempty"`
	Executables map[string]ExecInfo `json:"executables"`
}

// ExecInfo records the resolved location of a single executable.
type ExecInfo struct {
	Present bool   `json:"present"`
	Path    string `json:"path,omitempty"`
	Error   string `json:"error,omitempty"`
}

// SystemInfo captures locale/timezone/filesystem traits.
type SystemInfo struct {
	Timezone        string `json:"timezone,omitempty"`
	TZEnv           string `json:"tz_env,omitempty"`
	Locale          string `json:"locale,omitempty"`
	Lang            string `json:"lang,omitempty"`
	CaseSensitiveFS *bool  `json:"case_sensitive_fs,omitempty"`
}

// DockerInfo captures non-invasive Docker state (no containers are started).
type DockerInfo struct {
	Runtime      VersionInfo           `json:"runtime"`
	Compose      VersionInfo           `json:"compose"`
	ComposeFiles []string              `json:"compose_files,omitempty"`
	Images       []DockerImageInfo     `json:"images,omitempty"`
	Containers   []DockerContainerInfo `json:"containers,omitempty"`
}

// DockerImageInfo is a locally available image.
type DockerImageInfo struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	ID         string `json:"id,omitempty"`
}

// DockerContainerInfo is a running container.
type DockerContainerInfo struct {
	Name   string   `json:"name"`
	Image  string   `json:"image"`
	Status string   `json:"status"`
	Ports  []string `json:"ports,omitempty"`
}

// ProbeResults holds the outcome of active probes (only populated when probes
// run; omitted from passive snapshots).
type ProbeResults struct {
	TCP   []TCPProbeResult  `json:"tcp,omitempty"`
	DNS   []DNSProbeResult  `json:"dns,omitempty"`
	HTTP  []HTTPProbeResult `json:"http,omitempty"`
	TLS   []TLSProbeResult  `json:"tls,omitempty"`
	Proxy ProxyInfo         `json:"proxy,omitempty"`
	Ran   bool              `json:"ran"`
}

// TCPProbeResult records reachability of a TCP service.
type TCPProbeResult struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Port      int    `json:"port"`
	Required  bool   `json:"required"`
	Reachable bool   `json:"reachable"`
	Error     string `json:"error,omitempty"`
}

// DNSProbeResult records a DNS lookup outcome.
type DNSProbeResult struct {
	Host     string `json:"host"`
	Resolved bool   `json:"resolved"`
	Error    string `json:"error,omitempty"`
}

// HTTPProbeResult records an HTTP endpoint probe. Response bodies are never
// stored.
type HTTPProbeResult struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	Method         string `json:"method"`
	Required       bool   `json:"required"`
	ExpectedStatus int    `json:"expected_status,omitempty"`
	GotStatus      int    `json:"got_status,omitempty"`
	OK             bool   `json:"ok"`
	Error          string `json:"error,omitempty"`
}

// TLSProbeResult records a TLS handshake/certificate probe.
type TLSProbeResult struct {
	Name         string `json:"name"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	ServerName   string `json:"server_name,omitempty"`
	Required     bool   `json:"required"`
	HandshakeOK  bool   `json:"handshake_ok"`
	HostnameOK   bool   `json:"hostname_ok"`
	Issuer       string `json:"issuer,omitempty"`
	NotAfter     string `json:"not_after,omitempty"`
	DaysToExpiry int    `json:"days_to_expiry,omitempty"`
	Expired      bool   `json:"expired"`
	Error        string `json:"error,omitempty"`
}

// ProxyInfo records presence (not values) of proxy environment variables.
type ProxyInfo struct {
	Captured bool            `json:"captured"`
	Vars     map[string]bool `json:"vars,omitempty"` // var name -> set/unset
}
