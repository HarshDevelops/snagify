package capture

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

// fakeGitInit creates a minimal git repo in `dir` so envTracked has something
// to talk to. The caller is responsible for creating + committing files.
func fakeGitInit(t *testing.T, dir string) {
	t.Helper()
	if !gitAvailable() {
		t.Skip("git not available")
	}
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "config", "user.email", "t@example.com")
	runGit(t, dir, "config", "user.name", "test")
}

func TestScanEnvSafety_TrackedInGit(t *testing.T) {
	dir := t.TempDir()
	fakeGitInit(t, dir)
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SECRET=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", ".env")
	runGit(t, dir, "commit", "-q", "-m", "add env")

	safety := scanEnvSafety(dir, false)
	if !safety.Tracked {
		t.Errorf("expected Tracked=true; flags: %+v", safety)
	}
}

func TestScanEnvSafety_Untracked(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("SECRET=x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	// No git init.
	safety := scanEnvSafety(dir, false)
	if safety.Tracked {
		t.Errorf("expected Tracked=false outside a git repo; got %+v", safety)
	}
}

func TestScanEnvSafety_Gitignored_TopLevel(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".env\n*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("X=y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, false)
	if !safety.Gitignored {
		t.Errorf("expected Gitignored=true; got %+v", safety)
	}
}

func TestScanEnvSafety_Gitignored_NotPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("*.log\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, false)
	if safety.Gitignored {
		t.Errorf("expected Gitignored=false when no .env ignore; got %+v", safety)
	}
}

func TestScanEnvSafety_WeakMode_GroupReadable(t *testing.T) {
	if os.Getenv("GOOS") == "windows" {
		t.Skip("unix permission bits aren't meaningful on Windows")
	}
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("X=y\n"), 0o640); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, false)
	if !safety.WeakMode {
		t.Errorf("expected WeakMode=true at 0640; got %+v", safety)
	}
}

func TestScanEnvSafety_WeakMode_RestrictiveOK(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("X=y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, false)
	if safety.WeakMode {
		t.Errorf("expected WeakMode=false at 0600; got %+v", safety)
	}
}

func TestScanEnvSafety_WeakSecrets_KnownShapes(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	// Test fixtures are constructed programmatically so the literal
	// patterns don't appear in committed source — keeps GitHub's
	// secret-scanner quiet. Each value here matches the shape of a
	// known provider's secret format (and so must be flagged by
	// classifyWeak), but no real token is ever embedded.
	ghPrefix := "ghp_"                                   // GitHub PAT prefix
	ghBody := "aaaBBBcccDDDeeeFFF111222333444xx5566"    // exactly 36 chars after ghp_
	ghPrefix = ghPrefix[:4]                              // collapse to keep the literal off-disk
	ghToken := ghPrefix + ghBody
	slackPrefix := "xoxb-"                               // Slack token prefix
	slackBody := "12345678901-1234567890123-" + "abcdefghijklmnop" + "qrstuvwxyzAB"
	slackPrefix = slackPrefix[:5]                        // ditto
	slackToken := slackPrefix + slackBody

	content := "AWS_KEY=AKIAIOSFODNN7EXAMPLE\n" +
		"GH_TOKEN=" + ghToken + "\n" +
		"SLACK=" + slackToken + "\n" +
		"NORMAL=some_user_value\n"
	if err := os.WriteFile(envPath, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, true)

	got := map[string]string{}
	for _, h := range safety.WeakSecrets {
		got[h.Key] = h.Reason
	}

	want := map[string]string{
		"AWS_KEY": "aws-access-key",
		"GH_TOKEN": "github-pat",
		"SLACK":   "slack-token",
	}
	for k, wantReason := range want {
		if got[k] != wantReason {
			t.Errorf("key %q: got reason %q, want %q", k, got[k], wantReason)
		}
	}
	if _, ok := got["NORMAL"]; ok {
		t.Errorf("NORMAL should not be flagged; hits=%v", got)
	}
}

func TestScanEnvSafety_WeakSecrets_Entropy(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	// A 32-char base64-looking string with high entropy.
	high := "Aa1Bb2Cc3Dd4Ee5Ff6Gg7Hh8Ii9Jj0Kk1"
	if err := os.WriteFile(envPath, []byte("RANDOM="+high+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, true)
	if len(safety.WeakSecrets) != 1 || safety.WeakSecrets[0].Reason != "high-entropy-blob" {
		t.Errorf("expected one high-entropy hit; got %+v", safety.WeakSecrets)
	}
}

func TestScanEnvSafety_WeakSecrets_DisabledByDefault(t *testing.T) {
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("AWS=AKIAIOSFODNN7EXAMPLE\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	safety := scanEnvSafety(dir, false)
	if len(safety.WeakSecrets) != 0 {
		t.Errorf("value scan must be opt-in; got %+v", safety.WeakSecrets)
	}
}

func TestShannon_KnownValues(t *testing.T) {
	if math.Abs(shannon("aaaa")-0) > 1e-9 {
		t.Errorf("shannon(\"aaaa\") should be 0 (all same char)")
	}
	if math.Abs(shannon("abcd")-2.0) > 1e-9 {
		t.Errorf("shannon(\"abcd\") should be 2.0, got %v", shannon("abcd"))
	}
}

func TestUnquoteDotenv(t *testing.T) {
	cases := map[string]string{
		`"hello"`:       "hello",
		`'single-quote'`: "single-quote",
		`"unterminated`:  `"unterminated`,
		"plain":          "plain",
		"":               "",
		`""`:             "",
	}
	for in, want := range cases {
		if got := unquoteDotenv(in); got != want {
			t.Errorf("unquoteDotenv(%q)=%q, want %q", in, got, want)
		}
	}
}
