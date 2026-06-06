package capture

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/HarshDevelops/snagify/internal/model"
)

// manifestFiles are the marker files used to detect a project root. Order is
// significant only for naming heuristics, not detection.
var manifestFiles = []string{
	"package.json",
	"go.mod",
	"pom.xml",
	"build.gradle",
	"build.gradle.kts",
	"requirements.txt",
	"pyproject.toml",
	"Cargo.toml",
	"Gemfile",
	"composer.json",
}

// detectProject walks up from start looking for a directory containing any
// known manifest. If override is non-empty it is used directly. Returns the
// populated ProjectInfo; Root falls back to start when nothing is found.
func detectProject(start, override string) model.ProjectInfo {
	if override != "" {
		return describeProject(override)
	}

	dir := start
	for {
		if manifests := manifestsIn(dir); len(manifests) > 0 {
			return buildProjectInfo(dir, manifests)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached filesystem root
		}
		dir = parent
	}

	// No manifest found anywhere; report the starting directory.
	return model.ProjectInfo{Root: start}
}

// describeProject builds info for an explicitly provided root.
func describeProject(root string) model.ProjectInfo {
	return buildProjectInfo(root, manifestsIn(root))
}

func buildProjectInfo(dir string, manifests []string) model.ProjectInfo {
	return model.ProjectInfo{
		Root:      dir,
		Manifests: manifests,
		Name:      projectName(dir, manifests),
	}
}

// manifestsIn returns the manifest files present directly in dir.
func manifestsIn(dir string) []string {
	var found []string
	for _, m := range manifestFiles {
		if _, err := os.Stat(filepath.Join(dir, m)); err == nil {
			found = append(found, m)
		}
	}
	return found
}

// projectName derives a name from package.json, else uses the directory name.
func projectName(dir string, manifests []string) string {
	for _, m := range manifests {
		if m == "package.json" {
			if name := nameFromPackageJSON(filepath.Join(dir, m)); name != "" {
				return name
			}
		}
	}
	return filepath.Base(dir)
}

func nameFromPackageJSON(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pkg struct {
		Name string `json:"name"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return ""
	}
	return pkg.Name
}
