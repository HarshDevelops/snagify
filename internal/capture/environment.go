package capture

import (
	"runtime"

	"github.com/HarshDevelops/snagify/internal/model"
)

// captureEnvironment gathers OS, arch, OS version, and git version.
func captureEnvironment() model.Environment {
	env := model.Environment{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	if v := probe("git", "--version"); v.Present {
		env.GitVersion = v.Version
	}

	env.OSVersion = osVersion()
	return env
}

// osVersion returns a best-effort human OS version string per platform.
func osVersion() string {
	switch runtime.GOOS {
	case "darwin":
		if out, ok := runCmd("sw_vers", "-productVersion"); ok {
			return out
		}
	case "linux":
		// uname -r is universally available; distro detail lives in os-release.
		if out, ok := runCmd("uname", "-r"); ok {
			return out
		}
	case "windows":
		if out, ok := runCmd("cmd", "/c", "ver"); ok {
			return out
		}
	}
	return ""
}
