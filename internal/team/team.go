// Package team aggregates many snapshots against a baseline into a drift
// summary for a whole team.
package team

import (
	"sort"
	"strings"

	"github.com/harshdevelops/snagify/internal/diff"
	"github.com/harshdevelops/snagify/internal/model"
)

// MachineResult is one teammate's comparison against the baseline.
type MachineResult struct {
	Name     string   `json:"name"`
	Blockers []string `json:"blockers"`
	// Cells maps a column key (e.g. "Java", "Env", "Ports") to a status string.
	Cells map[string]string `json:"cells"`
	OK    bool              `json:"ok"`
}

// Summary is the aggregated team view.
type Summary struct {
	Columns        []string        `json:"columns"`
	Machines       []MachineResult `json:"machines"`
	CommonBlockers []BlockerCount  `json:"common_blockers"`
}

// BlockerCount tallies how many machines share a blocker.
type BlockerCount struct {
	Blocker string `json:"blocker"`
	Count   int    `json:"count"`
}

// Input pairs a machine display name with its snapshot.
type Input struct {
	Name     string
	Snapshot model.Snapshot
}

// Build compares each input snapshot against the baseline and produces a
// summary. Column set is the union of runtimes that differ on any machine,
// plus Env and Ports when relevant.
func Build(baseline model.Snapshot, inputs []Input) Summary {
	runtimeCols := map[string]bool{}
	var machines []MachineResult
	blockerTally := map[string]int{}

	for _, in := range inputs {
		res := diff.Compare(baseline, in.Snapshot, "baseline", in.Name)
		diff.ApplyProjectContext(&res, baseline, in.Snapshot)

		cells := map[string]string{}
		envIssue := false
		portIssue := false

		for _, e := range res.Entries {
			switch e.Category {
			case "Runtime":
				runtimeCols[e.Name] = true
				cells[e.Name] = cellFor(e)
			case "Env":
				envIssue = true
			case "Port":
				portIssue = true
			}
		}
		if envIssue {
			cells["Env"] = "❌"
		}
		if portIssue {
			cells["Ports"] = "❌"
		}

		mr := MachineResult{
			Name:     in.Name,
			Blockers: res.Blockers(),
			Cells:    cells,
			OK:       !res.HasDifferences(),
		}
		machines = append(machines, mr)

		for _, b := range res.Blockers() {
			blockerTally[normalizeBlocker(b)]++
		}
	}

	// Build ordered column list: runtimes (sorted) + Env + Ports.
	var cols []string
	rts := make([]string, 0, len(runtimeCols))
	for c := range runtimeCols {
		rts = append(rts, c)
	}
	sort.Strings(rts)
	cols = append(cols, rts...)
	cols = append(cols, "Env", "Ports")

	// Fill ok markers for machines that had no issue in a column.
	for i := range machines {
		for _, c := range cols {
			if _, ok := machines[i].Cells[c]; !ok {
				machines[i].Cells[c] = "ok"
			}
		}
	}

	return Summary{
		Columns:        cols,
		Machines:       machines,
		CommonBlockers: rankBlockers(blockerTally),
	}
}

func cellFor(e diff.Entry) string {
	// Prefer the teammate's value (B) for the cell, marked failing.
	v := e.B
	if v == "" {
		v = "?"
	}
	return v + " ❌"
}

func rankBlockers(tally map[string]int) []BlockerCount {
	out := make([]BlockerCount, 0, len(tally))
	for b, c := range tally {
		out = append(out, BlockerCount{Blocker: b, Count: c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Blocker < out[j].Blocker
	})
	return out
}

// normalizeBlocker collapses machine-specific blocker phrasing into a stable
// key so counts aggregate across machines.
func normalizeBlocker(b string) string {
	// diff blockers are phrased like "Maven is required by this project but
	// missing on <name>"; strip the trailing machine name for tallying.
	if i := strings.Index(b, " on "); i >= 0 {
		return strings.TrimSpace(b[:i])
	}
	return b
}
