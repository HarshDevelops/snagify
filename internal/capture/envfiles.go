package capture

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// captureEnvFiles checks for .env.example and .env in root, and reports keys
// declared in .env.example that are absent from .env.
func captureEnvFiles(root string) model.EnvFiles {
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

	return model.EnvFiles{
		EnvExampleExists: exampleExists,
		EnvExists:        envExists,
		MissingKeys:      missing,
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
