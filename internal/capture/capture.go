package capture

import (
	"os"
	"time"

	"github.com/harshdevelops/snagify/internal/model"
)

// Options configures a capture run.
type Options struct {
	// ProjectRoot overrides project auto-detection when non-empty.
	ProjectRoot string
}

// Capture builds a full Snapshot of the current machine and project.
func Capture(opts Options) (model.Snapshot, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return model.Snapshot{}, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	project := detectProject(cwd, opts.ProjectRoot)

	git := captureGit(project.Root)
	pathInfo := capturePath()
	system := captureSystem()
	docker := captureDocker(project.Root)

	return model.Snapshot{
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
		Hostname:    hostname,
		Project:     project,
		Environment: captureEnvironment(),
		Runtimes:    captureRuntimes(),
		Services:    capturePorts(),
		EnvFiles:    captureEnvFiles(project.Root),
		Git:         &git,
		Path:        &pathInfo,
		System:      &system,
		Docker:      &docker,
	}, nil
}
