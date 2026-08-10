package capture

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// captureEnvFiles checks for .env.example and .env in root, and reports keys
// declared in .env.example that are absent from .env. The safety pass always
// runs for the non-secret checks (tracked-in-git, gitignore coverage, file
// mode); the weak-secret value scan only runs when scanValues is true.
func captureEnvFiles(root string, scanValues bool) model.EnvFiles {
	examplePath := filepath.Join(root, ".env.example")
	envPath := filepath.Join(root, ".env")

	exampleKeys, exampleExists := readEnvKeys(examplePath)
	envKeys, envExists := readEnvKeys(envPath)

	var missing []string
	if exampleExists && envExists {
		for _, k := range exampleKeys {
			if !contains(envKeys, k) {
				missing = append(missing, k)
			}
		}
	} else if exampleExists && !envExists {
		// No .env at all: every example key is effectively missing.
		missing = append(missing, exampleKeys...)
	}

	safety := scanEnvSafety(root, scanValues)

	return model.EnvFiles{
		EnvExampleExists: exampleExists,
		EnvExists:        envExists,
		MissingKeys:      missing,
		Safety:           safety,
	}
}

// readEnvKeys returns the KEY names declared in a dotenv-style file and whether
// the file exists. Values are intentionally ignored — only presence matters.
func readEnvKeys(path string) ([]string, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var keys []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.Index(line, "=")
		if eq <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:eq])
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys, true
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
