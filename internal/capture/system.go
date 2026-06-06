package capture

import (
	"os"
	"path/filepath"
	"time"

	"github.com/harshdevelops/snagify/internal/model"
)

// captureSystem gathers timezone, locale, and filesystem case-sensitivity.
func captureSystem() model.SystemInfo {
	s := model.SystemInfo{
		Timezone: time.Local.String(),
		TZEnv:    os.Getenv("TZ"),
		Lang:     os.Getenv("LANG"),
	}

	// Locale: prefer LC_ALL, then LC_CTYPE, then LANG.
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(key); v != "" {
			s.Locale = v
			break
		}
	}

	if cs, ok := detectCaseSensitiveFS(); ok {
		s.CaseSensitiveFS = &cs
	}

	return s
}

// detectCaseSensitiveFS creates a temp file and checks whether its uppercase
// name resolves to the same file. It always cleans up. The second return is
// false when the check could not be performed.
func detectCaseSensitiveFS() (bool, bool) {
	dir, err := os.MkdirTemp(os.TempDir(), "snagify-case")
	if err != nil {
		return false, false
	}
	defer os.RemoveAll(dir)

	lower := filepath.Join(dir, "snagify-case-test")
	if err := os.WriteFile(lower, []byte("x"), 0o644); err != nil {
		return false, false
	}

	upper := filepath.Join(dir, "SNAGIFY-CASE-TEST")
	if _, err := os.Stat(upper); err == nil {
		// Uppercase resolves to the same file => case-insensitive FS.
		return false, true
	}
	// Uppercase does not resolve => case-sensitive FS.
	return true, true
}
