package model

// EnvSafety records safety-relevant signals about the project's .env file.
// These never include secret values; WeakSecrets carries only the key name
// and the reason a value was flagged.
type EnvSafety struct {
	// Tracked is true when `.env` is tracked in git. Tracking an .env file
	// (which holds live secrets) is almost always a mistake.
	Tracked bool `json:"tracked,omitempty"`
	// Gitignored is true when `.env` appears in any .gitignore in or above
	// the project root.
	Gitignored bool `json:"gitignored,omitempty"`
	// WeakMode is true when `.env` exists and is readable by group or other
	// (mode & 0o077 != 0). Unix only; false on Windows.
	WeakMode bool `json:"weak_mode,omitempty"`
	// WeakSecrets lists the env keys whose values match well-known secret
	// shapes. Values are never persisted.
	WeakSecrets []SecretHit `json:"weak_secrets,omitempty"`
	// EnvFile is the absolute path of the .env that was scanned. Empty if
	// no .env exists.
	EnvFile string `json:"env_file,omitempty"`
}

// SecretHit reports a single .env key whose value matched a known secret
// shape or exceeded an entropy threshold. The Value field is intentionally
// empty; populating it would re-introduce the very leak this report is
// trying to surface.
type SecretHit struct {
	Key    string `json:"key"`
	Reason string `json:"reason"`
}
