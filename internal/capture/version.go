package capture

import "regexp"

// semverRe extracts the first dotted version-like token from a string,
// e.g. "v20.11.1" -> "20.11.1", "go version go1.23.2 darwin/arm64" -> "1.23.2".
var semverRe = regexp.MustCompile(`(\d+(?:\.\d+){1,3})`)

// parseVersion pulls a clean version number out of arbitrary --version output.
func parseVersion(raw string) string {
	if m := semverRe.FindString(raw); m != "" {
		return m
	}
	return raw
}
