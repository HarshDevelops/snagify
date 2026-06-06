package diff

import (
	"fmt"

	"github.com/HarshDevelops/snagify/internal/model"
)

// projectRuntimes maps a manifest file to the runtimes it implies are
// required. Used to decide whether a missing runtime is a likely blocker.
var projectRuntimes = map[string][]string{
	"package.json":     {"Node"},
	"go.mod":           {"Go"},
	"pom.xml":          {"Java", "Maven"},
	"build.gradle":     {"Java", "Gradle"},
	"build.gradle.kts": {"Java", "Gradle"},
	"requirements.txt": {"Python"},
	"pyproject.toml":   {"Python"},
	"Cargo.toml":       {"Rust", "Cargo"},
}

// diffRuntimes compares every runtime across the two machines.
func diffRuntimes(a, b model.Runtimes) []Entry {
	var entries []Entry
	for _, f := range model.RuntimeFields {
		va, vb := f.Get(a), f.Get(b)
		entry, differ := compareRuntime(f.Name, va, vb)
		if differ {
			entries = append(entries, entry)
		}
	}
	return entries
}

// compareRuntime produces an Entry when two VersionInfos differ, classifying
// severity. The required-runtime/blocker upgrade happens in a later pass that
// knows the project type.
func compareRuntime(name string, a, b model.VersionInfo) (Entry, bool) {
	switch {
	case a.Present && !b.Present:
		return Entry{
			Category: "Runtime", Name: name,
			A: a.Version, B: "missing", Severity: Difference,
		}, true
	case !a.Present && b.Present:
		return Entry{
			Category: "Runtime", Name: name,
			A: "missing", B: b.Version, Severity: Difference,
		}, true
	case !a.Present && !b.Present:
		return Entry{}, false
	case a.Version != b.Version:
		sev := Difference
		if majorDiffers(a.Version, b.Version) {
			sev = Critical
		}
		e := Entry{
			Category: "Runtime", Name: name,
			A: a.Version, B: b.Version, Severity: sev,
		}
		if sev == Critical {
			e.Blocker = fmt.Sprintf("%s major version mismatch (%s vs %s)", name, a.Version, b.Version)
		}
		return e, true
	default:
		return Entry{}, false
	}
}

// ApplyProjectContext upgrades missing-runtime differences to critical blockers
// when the project's manifests imply that runtime is required. Both snapshots'
// manifests are considered so the requirement is detected from either side.
func ApplyProjectContext(res *Result, a, b model.Snapshot) {
	required := requiredRuntimes(a.Project.Manifests, b.Project.Manifests)
	for i := range res.Entries {
		e := &res.Entries[i]
		if e.Category != "Runtime" || !required[e.Name] {
			continue
		}
		if e.A == "missing" || e.B == "missing" {
			e.Severity = Critical
			who := res.NameB
			if e.A == "missing" {
				who = res.NameA
			}
			e.Blocker = fmt.Sprintf("%s is required by this project but missing on %s", e.Name, who)
		}
	}
}

func requiredRuntimes(manifestSets ...[]string) map[string]bool {
	req := map[string]bool{}
	for _, set := range manifestSets {
		for _, m := range set {
			for _, rt := range projectRuntimes[m] {
				req[rt] = true
			}
		}
	}
	return req
}
