package capture

import "github.com/HarshDevelops/snagify/internal/model"

// probe runs `bin <arg>` and returns a populated VersionInfo. It records the
// resolved binary path and parses a clean version from the output.
func probe(bin, arg string) model.VersionInfo {
	path := lookPath(bin)
	if path == "" {
		return model.VersionInfo{Present: false}
	}
	out, ok := runCmd(bin, arg)
	if !ok {
		// Binary exists but version probe failed; still report present.
		return model.VersionInfo{Present: true, Path: path}
	}
	return model.VersionInfo{
		Version: parseVersion(out),
		Present: true,
		Path:    path,
	}
}

// probePython prefers python3 but falls back to python.
func probePython() model.VersionInfo {
	if v := probe("python3", "--version"); v.Present {
		return v
	}
	return probe("python", "--version")
}

// captureRuntimes detects all known runtimes on the current machine.
func captureRuntimes() model.Runtimes {
	return model.Runtimes{
		Node:   probe("node", "--version"),
		NPM:    probe("npm", "--version"),
		PNPM:   probe("pnpm", "--version"),
		Yarn:   probe("yarn", "--version"),
		Python: probePython(),
		Pip:    probe("pip3", "--version"),
		UV:     probe("uv", "--version"),
		Java:   probe("java", "-version"),
		Maven:  probe("mvn", "--version"),
		Gradle: probe("gradle", "--version"),
		Go:     probe("go", "version"),
		Rust:   probe("rustc", "--version"),
		Cargo:  probe("cargo", "--version"),
		Docker: probe("docker", "--version"),
	}
}
