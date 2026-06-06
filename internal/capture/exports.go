package capture

// IsPortListening reports whether something is accepting TCP connections on
// the given port on localhost. Exported for the v0.2 `check` command, which
// must probe arbitrary ports declared in .snagify.yaml (not just the common
// dev ports captured in a snapshot).
func IsPortListening(port int) bool {
	return isListening(port)
}

// ReadEnvKeys returns the KEY names declared in a dotenv-style file and whether
// the file exists. Values are never returned — only key presence is exposed.
func ReadEnvKeys(path string) ([]string, bool) {
	return readEnvKeys(path)
}
