// Package baseline creates sanitized, shareable known-good snapshots and
// compares a local machine against them.
package baseline

import (
	"path/filepath"

	"github.com/harshdevelops/snagify/internal/diff"
	"github.com/harshdevelops/snagify/internal/model"
	"github.com/harshdevelops/snagify/internal/report"
)

// Sanitize returns a copy of snap safe to commit/share:
//   - hostname removed
//   - absolute project root removed (only the base name kept)
//
// It never touches env values or secrets because the snapshot model only ever
// stores env key names, never values.
func Sanitize(snap model.Snapshot) model.Snapshot {
	out := snap
	out.Hostname = ""

	name := snap.Project.Name
	if name == "" && snap.Project.Root != "" {
		name = filepath.Base(snap.Project.Root)
	}
	out.Project = model.ProjectInfo{
		Root:      "", // never store an absolute local path in a shareable baseline
		Name:      name,
		Manifests: snap.Project.Manifests,
	}
	return out
}

// Compare diffs the local snapshot against a baseline and returns a report
// using baseline/yours language. It reuses the v0.1 diff engine for the actual
// comparison, then maps the result into the v0.2 report model.
func Compare(baseline, current model.Snapshot) report.Report {
	res := diff.Compare(baseline, current, "baseline", "yours")
	diff.ApplyProjectContext(&res, baseline, current)

	r := report.Report{
		Project:    projectName(baseline, current),
		FoundLabel: "baseline",
		ExpLabel:   "yours",
	}

	for _, e := range res.Entries {
		r.Add(report.Item{
			Category: e.Category,
			Name:     e.Name,
			Found:    e.A, // baseline value
			Expected: e.B, // local value
			Severity: mapSeverity(e.Severity),
			Blocker:  e.Blocker,
		})
	}

	if r.Failed() {
		r.Status = "Your setup differs from baseline"
	} else if len(r.Items) > 0 {
		r.Status = "Your setup differs from baseline (minor)"
	} else {
		r.Status = "Your setup matches baseline"
	}
	return r
}

func mapSeverity(s diff.Severity) report.Severity {
	switch s {
	case diff.Critical:
		return report.Critical
	case diff.Difference:
		return report.Warning
	default:
		return report.Info
	}
}

func projectName(a, b model.Snapshot) string {
	if a.Project.Name != "" {
		return a.Project.Name
	}
	return b.Project.Name
}
