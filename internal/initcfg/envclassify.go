package initcfg

import (
	"bufio"
	"os"
	"strings"
)

// EnvClass is the classification of a single .env.example key.
type EnvClass int

const (
	EnvRequired    EnvClass = iota // missing == critical
	EnvRecommended                 // missing == warning
	EnvOptional                    // missing == silent
)

// rawLine is a parsed line from .env.example with its classification hint.
type rawLine struct {
	key     string
	value   string
	hint    EnvClass
	hintSet bool
}

// EnvEntry is one key from .env.example with its classification and raw default
// value. The raw value comes from .env.example only — never from .env.
type EnvEntry struct {
	Key   string
	Value string
	Class EnvClass
}

// ClassifyEnvExample parses a .env.example file and classifies every key.
// Priority: comment hint > disabled feature group > sensitive blank > default heuristics.
// Values are read from .env.example only — never from .env.
func ClassifyEnvExample(path string) ([]EnvEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// --- Pass 1: collect raw lines with their preceding comment hint ---
	var rawLines []rawLine
	var pendingHint EnvClass
	var pendingHintSet bool

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			pendingHintSet = false
			continue
		}

		if strings.HasPrefix(line, "#") {
			// Look for explicit classification hints.
			lower := strings.ToLower(line)
			switch {
			case strings.Contains(lower, "required"):
				pendingHint, pendingHintSet = EnvRequired, true
			case strings.Contains(lower, "recommended"):
				pendingHint, pendingHintSet = EnvRecommended, true
			case strings.Contains(lower, "optional"):
				pendingHint, pendingHintSet = EnvOptional, true
			}
			continue
		}

		line = strings.TrimPrefix(line, "export ")
		eq := strings.Index(line, "=")
		if eq <= 0 {
			pendingHintSet = false
			continue
		}
		key := strings.TrimSpace(line[:eq])
		val := strings.TrimSpace(line[eq+1:])
		if key == "" {
			pendingHintSet = false
			continue
		}

		rl := rawLine{key: key, value: val}
		if pendingHintSet {
			rl.hint = pendingHint
			rl.hintSet = true
		}
		rawLines = append(rawLines, rl)
		pendingHintSet = false
	}

	// --- Pass 2: identify disabled feature group prefixes ---
	disabledPrefixes := disabledFeaturePrefixes(rawLines)

	// --- Pass 3: classify each key ---
	entries := make([]EnvEntry, 0, len(rawLines))
	for _, rl := range rawLines {
		class := classify(rl.key, rl.value, rl.hint, rl.hintSet, disabledPrefixes)
		entries = append(entries, EnvEntry{Key: rl.key, Value: rl.value, Class: class})
	}
	return entries, nil
}

// disabledFeaturePrefixes scans lines for SOME_X_ENABLED=false and returns the
// corresponding prefixes (e.g. "SOME_X_") so the whole group can be optional.
func disabledFeaturePrefixes(lines []rawLine) map[string]bool {
	prefixes := map[string]bool{}
	for _, rl := range lines {
		if !isBoolFalse(rl.value) {
			continue
		}
		k := strings.ToUpper(rl.key)
		for _, suffix := range []string{"_ENABLED", "_ENABLE"} {
			if strings.HasSuffix(k, suffix) {
				prefix := k[:len(k)-len(suffix)+1] // keep the trailing _
				prefixes[prefix] = true
			}
		}
	}
	return prefixes
}

func isBoolFalse(v string) bool {
	return strings.EqualFold(strings.TrimSpace(v), "false") ||
		strings.TrimSpace(v) == "0"
}

// sensitiveTerms are name substrings that indicate a key is security-sensitive.
var sensitiveTerms = []string{
	"API_KEY", "TOKEN", "SECRET", "PASSWORD", "DATABASE_URL", "DB_URL",
	"ACCOUNT_ID", "CLIENT_ID", "CLIENT_SECRET", "AUTH", "CREDENTIAL",
}

// tuningTerms are name substrings that indicate a key is a tuning knob rather
// than a required secret.
var tuningTerms = []string{
	"MAX", "MIN", "RETRY", "TIMEOUT", "DELAY", "LIMIT", "THRESHOLD",
	"COUNT", "SIZE", "ENABLED", "DEBUG", "LOG_LEVEL", "MODE",
}

func classify(key, value string, hint EnvClass, hintSet bool, disabledPrefixes map[string]bool) EnvClass {
	// 1. Explicit comment hint wins.
	if hintSet {
		return hint
	}

	upper := strings.ToUpper(key)

	// 2. Disabled feature group => optional.
	for prefix := range disabledPrefixes {
		if strings.HasPrefix(upper, prefix) {
			return EnvOptional
		}
	}

	hasDefault := strings.TrimSpace(value) != ""

	// 3. Key with a concrete default:
	//    - tuning-like name => recommended
	//    - anything else with a default => recommended (not critical)
	if hasDefault {
		return EnvRecommended
	}

	// 4. Blank value + sensitive name => required.
	for _, term := range sensitiveTerms {
		if strings.Contains(upper, term) {
			return EnvRequired
		}
	}

	// 5. Blank value + tuning-like name => recommended.
	for _, term := range tuningTerms {
		if strings.Contains(upper, term) {
			return EnvRecommended
		}
	}

	// 6. Blank value, unknown name => recommended (not critical by default).
	return EnvRecommended
}
