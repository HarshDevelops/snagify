package initcfg

import (
	"os"
	"path/filepath"
	"testing"
)

// fixtureEnvExample is the generic test fixture from the spec.
const fixtureEnvExample = `CORE_API_KEY=
CLOUDFLARE_ACCOUNT_ID=
CLOUDFLARE_API_TOKEN=
NVIDIA_API_KEY=
MAX_RETRIES=3
RETRY_BASE_DELAY_MS=1500
RETRY_MAX_DELAY_MS=20000
MAX_EVIDENCE_ITEMS=12
CLOUDFLARE_BASE_URL=
FALLBACK_1_ENABLED=false
FALLBACK_1_NAME=nvidia
FALLBACK_1_BASE_URL=
FALLBACK_1_API_KEY=
FALLBACK_1_MODEL=
FALLBACK_2_ENABLED=false
FALLBACK_2_NAME=cloudflare
FALLBACK_2_BASE_URL=
FALLBACK_2_API_KEY=
FALLBACK_2_MODEL=
`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, ".env.example")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func classMap(entries []EnvEntry) map[string]EnvClass {
	m := map[string]EnvClass{}
	for _, e := range entries {
		m[e.Key] = e.Class
	}
	return m
}

func TestClassifyEnvExample_SpecFixture(t *testing.T) {
	p := writeFixture(t, fixtureEnvExample)
	entries, err := ClassifyEnvExample(p)
	if err != nil {
		t.Fatalf("ClassifyEnvExample: %v", err)
	}
	cm := classMap(entries)

	// Required
	for _, k := range []string{"CORE_API_KEY", "CLOUDFLARE_ACCOUNT_ID", "CLOUDFLARE_API_TOKEN", "NVIDIA_API_KEY"} {
		if cm[k] != EnvRequired {
			t.Errorf("%s should be required, got %v", k, cm[k])
		}
	}

	// Recommended (have concrete defaults or tuning names)
	for _, k := range []string{"MAX_RETRIES", "RETRY_BASE_DELAY_MS", "RETRY_MAX_DELAY_MS", "MAX_EVIDENCE_ITEMS"} {
		if cm[k] != EnvRecommended {
			t.Errorf("%s should be recommended, got %v", k, cm[k])
		}
	}
	// CLOUDFLARE_BASE_URL: blank + not obviously sensitive => recommended
	if cm["CLOUDFLARE_BASE_URL"] != EnvRecommended {
		t.Errorf("CLOUDFLARE_BASE_URL should be recommended, got %v", cm["CLOUDFLARE_BASE_URL"])
	}

	// Optional (disabled fallback groups)
	for _, k := range []string{
		"FALLBACK_1_ENABLED", "FALLBACK_1_NAME", "FALLBACK_1_BASE_URL", "FALLBACK_1_API_KEY", "FALLBACK_1_MODEL",
		"FALLBACK_2_ENABLED", "FALLBACK_2_NAME", "FALLBACK_2_BASE_URL", "FALLBACK_2_API_KEY", "FALLBACK_2_MODEL",
	} {
		if cm[k] != EnvOptional {
			t.Errorf("%s should be optional (disabled group), got %v", k, cm[k])
		}
	}

	// Non-negotiable: disabled fallback API keys must NOT be required.
	if cm["FALLBACK_1_API_KEY"] == EnvRequired {
		t.Error("FALLBACK_1_API_KEY must not be required when FALLBACK_1_ENABLED=false")
	}
	if cm["FALLBACK_2_API_KEY"] == EnvRequired {
		t.Error("FALLBACK_2_API_KEY must not be required when FALLBACK_2_ENABLED=false")
	}
}

func TestClassifyEnvExample_CommentHintOverrides(t *testing.T) {
	content := `
# optional
MAYBE_API_KEY=
# required
CORE_SECRET=
`
	p := writeFixture(t, content)
	entries, err := ClassifyEnvExample(p)
	if err != nil {
		t.Fatalf("ClassifyEnvExample: %v", err)
	}
	cm := classMap(entries)
	if cm["MAYBE_API_KEY"] != EnvOptional {
		t.Errorf("MAYBE_API_KEY should be optional via comment hint, got %v", cm["MAYBE_API_KEY"])
	}
	if cm["CORE_SECRET"] != EnvRequired {
		t.Errorf("CORE_SECRET should be required via comment hint, got %v", cm["CORE_SECRET"])
	}
}

func TestClassifyEnvExample_DefaultedTuningNotRequired(t *testing.T) {
	content := `MAX_RETRIES=3
TIMEOUT_MS=5000
FEATURE_ENABLED=false
`
	p := writeFixture(t, content)
	entries, err := ClassifyEnvExample(p)
	if err != nil {
		t.Fatalf("ClassifyEnvExample: %v", err)
	}
	for _, e := range entries {
		if e.Class == EnvRequired {
			t.Errorf("%s with default value should not be required", e.Key)
		}
	}
}

func TestClassifyEnvExample_SensitiveBlankRequired(t *testing.T) {
	content := `DATABASE_URL=
JWT_SECRET=
STRIPE_API_KEY=
`
	p := writeFixture(t, content)
	entries, err := ClassifyEnvExample(p)
	if err != nil {
		t.Fatalf("ClassifyEnvExample: %v", err)
	}
	cm := classMap(entries)
	for _, k := range []string{"DATABASE_URL", "JWT_SECRET", "STRIPE_API_KEY"} {
		if cm[k] != EnvRequired {
			t.Errorf("%s (sensitive, blank) should be required, got %v", k, cm[k])
		}
	}
}

func TestClassifyEnvExample_RequiredListStaysSmall(t *testing.T) {
	p := writeFixture(t, fixtureEnvExample)
	entries, _ := ClassifyEnvExample(p)
	var req int
	for _, e := range entries {
		if e.Class == EnvRequired {
			req++
		}
	}
	total := len(entries)
	if req > total/2 {
		t.Errorf("required list too large: %d/%d keys are required (expected small)", req, total)
	}
}
