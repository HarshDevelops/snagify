// Package diff compares two snapshots and produces ranked, categorized
// differences with plain-English blocker explanations.
package diff

import (
	"github.com/HarshDevelops/snagify/internal/model"
)

// Severity ranks a difference by how likely it is to break a setup.
type Severity int

const (
	// Critical differences are likely blockers.
	Critical Severity = iota
	// Difference is a real mismatch that may or may not matter.
	Difference
	// Info is contextual and rarely actionable.
	Info
)

func (s Severity) String() string {
	switch s {
	case Critical:
		return "critical"
	case Difference:
		return "difference"
	default:
		return "info"
	}
}

// Entry is a single compared field across machines A and B.
type Entry struct {
	Category string   `json:"category"` // e.g. "Runtime", "Port", "Env"
	Name     string   `json:"name"`
	A        string   `json:"a"`
	B        string   `json:"b"`
	Severity Severity `json:"severity"`
	Blocker  string   `json:"blocker,omitempty"` // plain-English reason, if any
}

// Result is the full comparison outcome.
type Result struct {
	ProjectName string  `json:"project"`
	NameA       string  `json:"name_a"`
	NameB       string  `json:"name_b"`
	Entries     []Entry `json:"entries"`
}

// Critical returns only entries flagged as likely blockers.
func (r Result) Critical() []Entry { return r.bySeverity(Critical) }

// Differences returns entries that differ but may be acceptable.
func (r Result) Differences() []Entry { return r.bySeverity(Difference) }

// Informational returns contextual entries.
func (r Result) Informational() []Entry { return r.bySeverity(Info) }

func (r Result) bySeverity(s Severity) []Entry {
	var out []Entry
	for _, e := range r.Entries {
		if e.Severity == s {
			out = append(out, e)
		}
	}
	return out
}

// Blockers returns the deduplicated plain-English blocker explanations.
func (r Result) Blockers() []string {
	var out []string
	seen := map[string]bool{}
	for _, e := range r.Entries {
		if e.Blocker != "" && !seen[e.Blocker] {
			seen[e.Blocker] = true
			out = append(out, e.Blocker)
		}
	}
	return out
}

// HasDifferences reports whether any non-informational entry exists.
func (r Result) HasDifferences() bool {
	for _, e := range r.Entries {
		if e.Severity != Info {
			return true
		}
	}
	return false
}

// Compare diffs two snapshots. nameA and nameB label the two machines in
// output (typically derived from the snapshot filenames or hostnames).
func Compare(a, b model.Snapshot, nameA, nameB string) Result {
	res := Result{
		ProjectName: projectLabel(a, b),
		NameA:       nameA,
		NameB:       nameB,
	}

	res.Entries = append(res.Entries, diffEnvironment(a.Environment, b.Environment)...)
	res.Entries = append(res.Entries, diffRuntimes(a.Runtimes, b.Runtimes)...)
	res.Entries = append(res.Entries, diffPorts(a.Services, b.Services)...)
	res.Entries = append(res.Entries, diffEnvFiles(a.EnvFiles, b.EnvFiles)...)

	return res
}

func projectLabel(a, b model.Snapshot) string {
	if a.Project.Name != "" {
		return a.Project.Name
	}
	if b.Project.Name != "" {
		return b.Project.Name
	}
	return "project"
}

func diffEnvironment(a, b model.Environment) []Entry {
	var entries []Entry
	if a.OS != b.OS {
		entries = append(entries, Entry{
			Category: "Environment", Name: "OS",
			A: a.OS, B: b.OS, Severity: Info,
		})
	}
	if a.Arch != b.Arch {
		entries = append(entries, Entry{
			Category: "Environment", Name: "Arch",
			A: a.Arch, B: b.Arch, Severity: Info,
		})
	}
	if a.GitVersion != b.GitVersion {
		entries = append(entries, Entry{
			Category: "Environment", Name: "git",
			A: orMissing(a.GitVersion), B: orMissing(b.GitVersion),
			Severity: Difference,
		})
	}
	return entries
}

func orMissing(s string) string {
	if s == "" {
		return "missing"
	}
	return s
}
