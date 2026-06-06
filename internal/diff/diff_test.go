package diff

import (
	"testing"

	"github.com/harshdevelops/snagify/internal/model"
)

func vi(version string) model.VersionInfo {
	return model.VersionInfo{Version: version, Present: true}
}

func absent() model.VersionInfo { return model.VersionInfo{Present: false} }

func TestMajorDiffers(t *testing.T) {
	cases := []struct {
		a, b string
		want bool
	}{
		{"18.0.0", "22.1.0", true},
		{"20.11.1", "20.12.0", false},
		{"17.0.9", "21.0.2", true},
		{"3.9", "3.9", false},
		{"unknown", "1.0", false},
	}
	for _, c := range cases {
		if got := majorDiffers(c.a, c.b); got != c.want {
			t.Errorf("majorDiffers(%q,%q) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

func TestCompare_MajorVersionIsCritical(t *testing.T) {
	a := model.Snapshot{Runtimes: model.Runtimes{Node: vi("18.0.0")}}
	b := model.Snapshot{Runtimes: model.Runtimes{Node: vi("22.0.0")}}

	res := Compare(a, b, "a", "b")
	crit := res.Critical()
	if len(crit) != 1 || crit[0].Name != "Node" {
		t.Fatalf("expected 1 critical Node entry, got %+v", crit)
	}
	if crit[0].Blocker == "" {
		t.Error("expected a blocker message for major version mismatch")
	}
}

func TestCompare_MinorVersionIsDifference(t *testing.T) {
	a := model.Snapshot{Runtimes: model.Runtimes{Node: vi("20.11.0")}}
	b := model.Snapshot{Runtimes: model.Runtimes{Node: vi("20.12.0")}}

	res := Compare(a, b, "a", "b")
	if len(res.Critical()) != 0 {
		t.Errorf("minor diff should not be critical: %+v", res.Critical())
	}
	if len(res.Differences()) != 1 {
		t.Errorf("expected 1 difference, got %+v", res.Differences())
	}
}

func TestApplyProjectContext_MissingRequiredRuntime(t *testing.T) {
	a := model.Snapshot{
		Project:  model.ProjectInfo{Manifests: []string{"pom.xml"}},
		Runtimes: model.Runtimes{Maven: vi("3.9.6")},
	}
	b := model.Snapshot{
		Project:  model.ProjectInfo{Manifests: []string{"pom.xml"}},
		Runtimes: model.Runtimes{Maven: absent()},
	}

	res := Compare(a, b, "harsh", "teammate")
	ApplyProjectContext(&res, a, b)

	var found bool
	for _, e := range res.Critical() {
		if e.Name == "Maven" && e.Blocker != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected Maven missing to be critical blocker, got %+v", res.Entries)
	}
}

func TestCompare_Identical_NoDiff(t *testing.T) {
	s := model.Snapshot{
		Environment: model.Environment{OS: "darwin", Arch: "arm64", GitVersion: "2.39.0"},
		Runtimes:    model.Runtimes{Go: vi("1.23.2")},
		Services:    model.Services{Ports: map[int]model.PortInfo{5432: {Port: 5432, Listening: true}}},
	}
	res := Compare(s, s, "a", "b")
	if res.HasDifferences() {
		t.Errorf("identical snapshots should have no differences, got %+v", res.Entries)
	}
}

func TestCompare_EnvKeysMissing(t *testing.T) {
	a := model.Snapshot{EnvFiles: model.EnvFiles{MissingKeys: nil}}
	b := model.Snapshot{EnvFiles: model.EnvFiles{MissingKeys: []string{"DATABASE_URL", "JWT_SECRET"}}}

	res := Compare(a, b, "a", "b")
	blockers := res.Blockers()
	if len(blockers) == 0 {
		t.Fatal("expected a blocker for missing env keys")
	}
}

func TestCompare_PortStateDiffers(t *testing.T) {
	a := model.Snapshot{Services: model.Services{Ports: map[int]model.PortInfo{
		5432: {Port: 5432, Service: "Postgres", Listening: true},
	}}}
	b := model.Snapshot{Services: model.Services{Ports: map[int]model.PortInfo{
		5432: {Port: 5432, Service: "Postgres", Listening: false},
	}}}

	res := Compare(a, b, "a", "b")
	if len(res.Differences()) != 1 {
		t.Fatalf("expected 1 port difference, got %+v", res.Differences())
	}
}
