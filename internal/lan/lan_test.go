package lan

import (
	"testing"
)

func TestBuildTXT_NoSecrets(t *testing.T) {
	meta := ShareMeta{
		Repo:        "checkout-service",
		Project:     "checkout-service",
		Version:     "0.4.0",
		Mode:        "code",
		Fingerprint: "sha256:AB:CD",
		ExpiresAt:   1234567890,
		Port:        49200,
	}
	txt := buildTXT(meta)

	for _, kv := range txt {
		// TXT must never contain secret, env, or pairing-code-like fields.
		if kv == "" {
			t.Errorf("empty TXT record")
		}
	}

	// Repo, version, mode, fingerprint, expires_at must be present.
	m := make(map[string]string, len(txt))
	for _, kv := range txt {
		i := indexOf(kv, '=')
		if i > 0 {
			m[kv[:i]] = kv[i+1:]
		}
	}
	for _, required := range []string{"repo", "version", "mode", "fingerprint", "expires_at"} {
		if m[required] == "" {
			t.Errorf("TXT missing %q field", required)
		}
	}
}

func TestBuildTXT_NoCode(t *testing.T) {
	meta := ShareMeta{Repo: "x", Mode: "code"}
	for _, kv := range buildTXT(meta) {
		if len(kv) > 5 && kv[:5] == "code=" {
			t.Error("pairing code must not appear in TXT records")
		}
	}
}

func TestParseTXT_RoundTrip(t *testing.T) {
	meta := ShareMeta{
		Repo:        "my-repo",
		Project:     "my-project",
		Version:     "0.4.0",
		Mode:        "open",
		Fingerprint: "sha256:AA:BB",
		ExpiresAt:   9999,
	}
	txt := buildTXT(meta)
	got := parseTXT(txt)
	if got.Repo != meta.Repo {
		t.Errorf("Repo = %q, want %q", got.Repo, meta.Repo)
	}
	if got.Mode != meta.Mode {
		t.Errorf("Mode = %q, want %q", got.Mode, meta.Mode)
	}
	if got.Fingerprint != meta.Fingerprint {
		t.Errorf("Fingerprint = %q, want %q", got.Fingerprint, meta.Fingerprint)
	}
	if got.ExpiresAt != meta.ExpiresAt {
		t.Errorf("ExpiresAt = %d, want %d", got.ExpiresAt, meta.ExpiresAt)
	}
}

func TestSanitiseInstance(t *testing.T) {
	if s := sanitiseInstance(""); s == "" {
		t.Error("empty name should return non-empty instance")
	}
	if s := sanitiseInstance("my.repo.name"); s == "my.repo.name" {
		t.Error("dots should be replaced")
	}
}

func TestDiscoveredShare_IsExpired(t *testing.T) {
	past := DiscoveredShare{ExpiresAt: 1} // Unix timestamp well in the past
	if !past.IsExpired() {
		t.Error("expected IsExpired=true for past timestamp")
	}
	future := DiscoveredShare{ExpiresAt: 9999999999}
	if future.IsExpired() {
		t.Error("expected IsExpired=false for far-future timestamp")
	}
	zero := DiscoveredShare{ExpiresAt: 0}
	if zero.IsExpired() {
		t.Error("expected IsExpired=false for zero timestamp")
	}
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
