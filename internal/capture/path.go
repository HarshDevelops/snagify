package capture

import (
	"os"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// trackedExecutables are the binaries whose resolved locations are captured.
var trackedExecutables = []string{
	"node", "npm", "pnpm", "yarn",
	"python", "python3", "pip", "uv",
	"java", "mvn", "gradle",
	"go", "rustc", "cargo",
	"docker", "git",
}

// capturePath records redacted PATH entries and resolved executable paths.
func capturePath() model.PathInfo {
	out := model.PathInfo{
		Executables: make(map[string]model.ExecInfo),
	}

	for _, dir := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		if dir == "" {
			continue
		}
		out.PathEntries = append(out.PathEntries, redactHome(dir))
	}

	for _, name := range trackedExecutables {
		p := lookPath(name)
		if p == "" {
			out.Executables[name] = model.ExecInfo{Present: false}
			continue
		}
		out.Executables[name] = model.ExecInfo{
			Present: true,
			Path:    redactHome(p),
		}
	}
	return out
}
