package capture

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/harshdevelops/snagify/internal/model"
)

// scanEnvSafety populates an EnvSafety report for the project's `.env`.
// The non-secret checks (tracked-in-git, gitignore, file mode) always run.
// The weak-secret value scan only runs when scanValues is true; reading
// secret values crosses the project's "values are never captured" line and
// must be explicitly opted into via the --scan-secrets flag.
func scanEnvSafety(root string, scanValues bool) *model.EnvSafety {
	envPath := filepath.Join(root, ".env")

	safety := &model.EnvSafety{}

	if _, err := os.Stat(envPath); err == nil {
		safety.EnvFile = envPath
		safety.WeakMode = checkWeakMode(envPath)
	}

	safety.Tracked = envTracked(root)
	safety.Gitignored = envGitignored(root)

	if scanValues && safety.EnvFile != "" {
		safety.WeakSecrets = scanWeakSecrets(envPath)
	}

	return safety
}

// envTracked reports whether `.env` is checked into git. We use
// `git ls-files .env` and consider any tracked result a critical finding;
// `git status` is too easy to misinterpret with submodules and partial
// checkouts.
func envTracked(root string) bool {
	out, ok := gitOut(root, "ls-files", "-z", "--", ".env")
	if !ok {
		return false
	}
	for _, line := range strings.Split(out, "\x00") {
		if line == ".env" {
			return true
		}
	}
	return false
}

// envGitignored reports whether any .gitignore in or above root mentions
// `.env`. We accept any match (`.env`, `*.env`, `.env*`) because the safest
// takeaway is "user has expressed an intent to ignore env-like files".
func envGitignored(root string) bool {
	dir := root
	for {
		candidate := filepath.Join(dir, ".gitignore")
		if _, err := os.Stat(candidate); err == nil {
			f, err := os.Open(candidate)
			if err != nil {
				return false
			}
			defer f.Close()
			scanner := bufio.NewScanner(f)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				if line == ".env" || strings.HasSuffix(line, "/.env") ||
					line == ".env*" || line == "*.env" {
					return true
				}
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return false
		}
		dir = parent
	}
}

// checkWeakMode reports whether the file's permission bits allow group/other
// to read it. Mode is taken from os.Stat — Windows always returns 0o644 and
// Unix permissions aren't enforced, so this is effectively a no-op on
// Windows.
func checkWeakMode(path string) bool {
	fi, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fi.Mode().Perm()&0o077 != 0
}

// weakSecretPatterns maps a textual reason to a compiled regex matching the
// value side of a `KEY=value` line. All patterns are well-known secret
// formats drawn from their respective providers' public documentation:
//   - AWS root keys:        docs.aws.amazon.com/IAM
//   - GitHub PATs:          docs.github.com/authentication
//   - Stripe:               stripe.com/docs/keys
//   - OpenAI project keys:  platform.openai.com/docs
//   - Slack tokens:         api.slack.com/authentication
//   - Google API keys:      cloud.google.com/api-keys
//   - JWT (header.payload.sig) - well-known shape, no claim validation.
var weakSecretPatterns = []struct {
	reason  string
	pattern *regexp.Regexp
}{
	{"aws-access-key", regexp.MustCompile(`\b(AKIA|ASIA)[0-9A-Z]{16}\b`)},
	{"github-pat", regexp.MustCompile(`\bghp_[A-Za-z0-9]{36}\b`)},
	{"stripe-live", regexp.MustCompile(`\bsk_live_[A-Za-z0-9]{20,}\b`)},
	{"openai-project-key", regexp.MustCompile(`\bsk-[A-Za-z0-9]{20,}T3BlbkFJ[A-Za-z0-9]{20,}\b`)},
	{"slack-token", regexp.MustCompile(`\bxox[baprs]-[A-Za-z0-9-]{10,}\b`)},
	{"google-api-key", regexp.MustCompile(`\bAIza[0-9A-Za-z\-_]{35}\b`)},
	{"jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{8,}\.eyJ[A-Za-z0-9_-]{8,}\.[A-Za-z0-9_-]{8,}\b`)},
}

// highEntropyThreshold is the Shannon-entropy bar applied to long values
// that don't match a known pattern. 4.5 bits/char over a string of at least
// 24 characters catches random base64/hex blobs while ignoring dictionary
// words.
const (
	highEntropyThreshold = 4.5
	highEntropyMinLength = 24
)

// scanWeakSecrets reports env keys whose VALUES match a known secret
// pattern OR exceed a high entropy bar. Values themselves are never
// persisted; only the key name and the reason are returned.
func scanWeakSecrets(envPath string) []model.SecretHit {
	f, err := os.Open(envPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024), 1<<16) // up to 64 KiB lines
	var hits []model.SecretHit
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
		value := unquoteDotenv(strings.TrimSpace(line[eq+1:]))
		if value == "" {
			continue
		}
		reason := classifyWeak(value)
		if reason == "" {
			continue
		}
		hits = append(hits, model.SecretHit{Key: key, Reason: reason})
	}
	return hits
}

// classifyWeak returns the reason a value looks like a leaked secret, or "".
// Tries known patterns first; falls back to entropy on long values.
func classifyWeak(value string) string {
	for _, p := range weakSecretPatterns {
		if p.pattern.MatchString(value) {
			return p.reason
		}
	}
	if len(value) >= highEntropyMinLength {
		if shannon(value) >= highEntropyThreshold {
			return "high-entropy-blob"
		}
	}
	return ""
}

// unquoteDotenv strips a single layer of " or ' from a value, matching the
// behaviour of `dotenv` parsers. Empty values still empty-quote cleanly.
func unquoteDotenv(s string) string {
	if len(s) < 2 {
		return s
	}
	if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
		return s[1 : len(s)-1]
	}
	return s
}

// shannon returns the Shannon entropy of s in bits per character.
func shannon(s string) float64 {
	if s == "" {
		return 0
	}
	counts := make(map[rune]int)
	for _, c := range s {
		counts[c]++
	}
	total := float64(len(s))
	var h float64
	for _, n := range counts {
		p := float64(n) / total
		h -= p * math.Log2(p)
	}
	return h
}
