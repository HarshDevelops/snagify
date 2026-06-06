package baseline

import (
	"strings"
	"testing"

	"github.com/harshdevelops/snagify/internal/model"
)

func TestSanitize_RemovesHostnameAndRoot(t *testing.T) {
	snap := model.Snapshot{
		Hostname: "harsh-laptop",
		Project: model.ProjectInfo{
			Root:      "/Users/harsh/dev/checkout-service",
			Name:      "checkout-service",
			Manifests: []string{"pom.xml"},
		},
		Runtimes: model.Runtimes{Java: model.VersionInfo{Version: "17.0.9", Present: true}},
	}

	s := Sanitize(snap)
	if s.Hostname != "" {
		t.Errorf("hostname not removed: %q", s.Hostname)
	}
	if s.Project.Root != "" {
		t.Errorf("absolute root not removed: %q", s.Project.Root)
	}
	if s.Project.Name != "checkout-service" {
		t.Errorf("project name lost: %q", s.Project.Name)
	}
	if s.Runtimes.Java.Version != "17.0.9" {
		t.Error("runtime version should be preserved")
	}
}

func TestSanitize_DerivesNameFromRootWhenMissing(t *testing.T) {
	snap := model.Snapshot{Project: model.ProjectInfo{Root: "/home/x/my-app"}}
	s := Sanitize(snap)
	if s.Project.Name != "my-app" {
		t.Errorf("name = %q, want my-app", s.Project.Name)
	}
}

func TestCompare_DiffersFromBaseline(t *testing.T) {
	base := model.Snapshot{
		Project:  model.ProjectInfo{Name: "checkout-service", Manifests: []string{"pom.xml"}},
		Runtimes: model.Runtimes{Java: model.VersionInfo{Version: "17.0.9", Present: true}, Maven: model.VersionInfo{Version: "3.9.6", Present: true}},
	}
	mine := model.Snapshot{
		Project:  model.ProjectInfo{Name: "checkout-service", Manifests: []string{"pom.xml"}},
		Runtimes: model.Runtimes{Java: model.VersionInfo{Version: "21.0.2", Present: true}, Maven: model.VersionInfo{Present: false}},
	}

	r := Compare(base, mine)
	if !r.Failed() {
		t.Fatal("expected failure: java major differs and maven missing")
	}
	if !strings.Contains(r.Status, "differs from baseline") {
		t.Errorf("status = %q, want 'differs from baseline'", r.Status)
	}
}

func TestCompare_Matches(t *testing.T) {
	s := model.Snapshot{
		Project:  model.ProjectInfo{Name: "app"},
		Runtimes: model.Runtimes{Go: model.VersionInfo{Version: "1.23.2", Present: true}},
	}
	r := Compare(s, s)
	if r.Failed() {
		t.Errorf("identical should not fail: %v", r.Blockers())
	}
}
