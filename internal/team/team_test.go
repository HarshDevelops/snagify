package team

import (
	"bytes"
	"strings"
	"testing"

	"github.com/harshdevelops/snagify/internal/model"
)

func vi(v string) model.VersionInfo { return model.VersionInfo{Version: v, Present: true} }

func TestBuild_SummarizesDriftAndBlockers(t *testing.T) {
	base := model.Snapshot{
		Project: model.ProjectInfo{Name: "checkout-service", Manifests: []string{"pom.xml"}},
		Runtimes: model.Runtimes{
			Java:   vi("17.0.9"),
			Maven:  vi("3.9.6"),
			Docker: vi("25.0.3"),
		},
	}

	mkSnap := func(java string, mavenPresent, dockerPresent bool) model.Snapshot {
		s := model.Snapshot{
			Project:  model.ProjectInfo{Name: "checkout-service", Manifests: []string{"pom.xml"}},
			Runtimes: model.Runtimes{Java: vi(java), Maven: vi("3.9.6"), Docker: vi("25.0.3")},
		}
		if !mavenPresent {
			s.Runtimes.Maven = model.VersionInfo{Present: false}
		}
		if !dockerPresent {
			s.Runtimes.Docker = model.VersionInfo{Present: false}
		}
		return s
	}

	inputs := []Input{
		{Name: "aman", Snapshot: mkSnap("21.0.2", false, true)}, // java mismatch + maven missing
		{Name: "priya", Snapshot: mkSnap("17.0.9", true, true)}, // ok
		{Name: "ravi", Snapshot: mkSnap("17.0.9", true, false)}, // docker missing
	}

	s := Build(base, inputs)

	if len(s.Machines) != 3 {
		t.Fatalf("expected 3 machines, got %d", len(s.Machines))
	}
	if len(s.CommonBlockers) == 0 {
		t.Fatal("expected common blockers tally")
	}

	// Renderers must run without error and not panic.
	var term, md bytes.Buffer
	Terminal(&term, s)
	Markdown(&md, s)
	if !strings.Contains(term.String(), "Team setup drift") {
		t.Error("terminal output missing header")
	}
	if !strings.Contains(md.String(), "| Name |") {
		t.Error("markdown output missing table header")
	}

	var jbuf bytes.Buffer
	if err := JSON(&jbuf, s); err != nil {
		t.Fatalf("json render: %v", err)
	}
}
