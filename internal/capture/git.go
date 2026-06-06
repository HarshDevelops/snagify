package capture

import (
	"strconv"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// captureGit gathers git state for the project root. It never crashes on git
// errors; absence of a repo yields Present=false.
func captureGit(root string) model.GitInfo {
	var g model.GitInfo

	inside, ok := gitOut(root, "rev-parse", "--is-inside-work-tree")
	if !ok || strings.TrimSpace(inside) != "true" {
		g.Present = false
		return g
	}
	g.Present = true

	if branch, ok := gitOut(root, "branch", "--show-current"); ok {
		g.Branch = strings.TrimSpace(branch)
	}
	if commit, ok := gitOut(root, "rev-parse", "--short", "HEAD"); ok {
		g.Commit = strings.TrimSpace(commit)
	}

	if status, ok := gitOut(root, "status", "--porcelain"); ok {
		for _, line := range strings.Split(status, "\n") {
			if line == "" {
				continue
			}
			if strings.HasPrefix(line, "??") {
				g.Untracked = true
			} else {
				g.Dirty = true
			}
		}
	}

	// Ahead/behind only when an upstream is configured.
	if counts, ok := gitOut(root, "rev-list", "--left-right", "--count", "@{upstream}...HEAD"); ok {
		fields := strings.Fields(strings.TrimSpace(counts))
		if len(fields) == 2 {
			g.Behind, _ = strconv.Atoi(fields[0])
			g.Ahead, _ = strconv.Atoi(fields[1])
		}
	}

	return g
}

// gitOut runs a git command in dir and returns trimmed output + success.
func gitOut(dir string, args ...string) (string, bool) {
	full := append([]string{"-C", dir}, args...)
	return runCmd("git", full...)
}
