package render

import (
	"encoding/json"
	"io"

	"github.com/harshdevelops/snagify/internal/diff"
)

// JSON writes the diff result as indented JSON to w.
func JSON(w io.Writer, r diff.Result) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(newJSONView(r))
}

// jsonView is a scripting-friendly projection of the diff result, with
// severities as strings and blockers surfaced as a top-level list.
type jsonView struct {
	Project  string      `json:"project"`
	A        string      `json:"a"`
	B        string      `json:"b"`
	HasDiff  bool        `json:"has_differences"`
	Blockers []string    `json:"blockers"`
	Entries  []jsonEntry `json:"entries"`
}

type jsonEntry struct {
	Category string `json:"category"`
	Name     string `json:"name"`
	A        string `json:"a"`
	B        string `json:"b"`
	Severity string `json:"severity"`
	Blocker  string `json:"blocker,omitempty"`
}

func newJSONView(r diff.Result) jsonView {
	v := jsonView{
		Project:  r.ProjectName,
		A:        r.NameA,
		B:        r.NameB,
		HasDiff:  r.HasDifferences(),
		Blockers: r.Blockers(),
	}
	if v.Blockers == nil {
		v.Blockers = []string{}
	}
	for _, e := range r.Entries {
		v.Entries = append(v.Entries, jsonEntry{
			Category: e.Category,
			Name:     e.Name,
			A:        e.A,
			B:        e.B,
			Severity: e.Severity.String(),
			Blocker:  e.Blocker,
		})
	}
	if v.Entries == nil {
		v.Entries = []jsonEntry{}
	}
	return v
}
