// Package plugin implements the subprocess + JSON-RPC plugin contract.
//
// Snagify plugins are *external* binaries — separate executables invoked
// via os/exec — that read one JSON object from stdin (the request) and
// emit one JSON object on stdout (the response). This sidesteps Go's
// in-process `plugin.Open` (Linux-only, brittle, no cross-compile) in
// favour of a contract that any language can implement.
//
// Wire format (v1):
//
//	request:  {"protocol":"snagify/plugin/v1","kind":"checks","checks":[<Spec>]}
//	response: {"protocol":"snagify/plugin/v1","results":[<Result>]}
//
// Plugin authors implement `kind: checks` first — it is the only kind
// Snagify currently invokes. New kinds (e.g. "probes", "captures") will
// be versioned alongside.
package plugin

// Protocol is the only protocol version Snagify speaks today. Plugins
// claiming any other value are rejected at the loader.
const Protocol = "snagify/plugin/v1"

// Kind is what the plugin knows how to do. Today only "checks" is wired
// into the loader. Other kinds (probes, captures) are reserved.
type Kind string

const (
	KindChecks Kind = "checks"
)

// Spec is one check the plugin should run. It's the same shape as the
// in-process customcheck.Spec; we copy it here so the plugin contract
// does not depend on internal Go packages.
type Spec struct {
	Name     string   `json:"name"`
	Run      []string `json:"run"`
	Severity string   `json:"severity,omitempty"`
	Message  string   `json:"message,omitempty"`
	Timeout  string   `json:"timeout,omitempty"`
	Workdir  string   `json:"workdir,omitempty"`
}

// Request is the JSON document the host sends on stdin.
type Request struct {
	Protocol string `json:"protocol"`
	Kind     Kind   `json:"kind"`
	Checks   []Spec `json:"checks"`
}

// Result is one outcome. Status is "ok", "warning", or "critical".
type Result struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Message  string `json:"message,omitempty"`
	ExitCode int    `json:"exit_code,omitempty"`
}

// Response is the JSON document the plugin emits on stdout.
type Response struct {
	Protocol string   `json:"protocol"`
	Results  []Result `json:"results"`
	// Error is set only when the plugin itself bailed before any
	// per-check execution. Per-check failures belong in Result.Status.
	Error string `json:"error,omitempty"`
}
