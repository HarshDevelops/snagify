package capture

import (
	"os"
	"strings"
)

// redactHome replaces the user's home directory prefix with "~" so captured
// paths never leak usernames or absolute home locations.
func redactHome(path string) string {
	if path == "" {
		return path
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return path
	}
	if path == home {
		return "~"
	}
	if strings.HasPrefix(path, home+string(os.PathSeparator)) {
		return "~" + path[len(home):]
	}
	return path
}
