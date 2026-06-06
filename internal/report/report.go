// Package report is a shared, render-agnostic result model used by the v0.2
// check and baseline commands. It intentionally mirrors the spec's
// "Critical blockers / Warnings / Info" structure and is distinct from the
// v0.1 diff.Result type so existing behavior is untouched.
package report

// Severity ranks an item.
type Severity int

const (
	// Critical items are blockers; their presence fails a check.
	Critical Severity = iota
	// Warning items differ but do not fail a check.
	Warning
	// Info items are contextual.
	Info
)

func (s Severity) String() string {
	switch s {
	case Critical:
		return "critical"
	case Warning:
		return "warning"
	default:
		return "info"
	}
}

// Item is a single compared row.
type Item struct {
	Category string   `json:"category"`
	Name     string   `json:"name"`
	Found    string   `json:"found"`
	Expected string   `json:"expected"`
	Severity Severity `json:"severity"`
	Blocker  string   `json:"blocker,omitempty"`
}

// Report is the full result of a check or baseline comparison.
type Report struct {
	Project    string `json:"project"`
	Status     string `json:"status"`
	FoundLabel string `json:"found_label"`
	ExpLabel   string `json:"expected_label"`
	Items      []Item `json:"items"`
}

// Add appends an item.
func (r *Report) Add(it Item) { r.Items = append(r.Items, it) }

// Critical returns critical items.
func (r Report) Critical() []Item { return r.bySeverity(Critical) }

// Warnings returns warning items.
func (r Report) Warnings() []Item { return r.bySeverity(Warning) }

// Informational returns info items.
func (r Report) Informational() []Item { return r.bySeverity(Info) }

func (r Report) bySeverity(s Severity) []Item {
	var out []Item
	for _, it := range r.Items {
		if it.Severity == s {
			out = append(out, it)
		}
	}
	return out
}

// Failed reports whether the report contains any critical item.
func (r Report) Failed() bool {
	for _, it := range r.Items {
		if it.Severity == Critical {
			return true
		}
	}
	return false
}

// Blockers returns the deduplicated plain-English blocker explanations.
func (r Report) Blockers() []string {
	var out []string
	seen := map[string]bool{}
	for _, it := range r.Items {
		if it.Blocker != "" && !seen[it.Blocker] {
			seen[it.Blocker] = true
			out = append(out, it.Blocker)
		}
	}
	return out
}
