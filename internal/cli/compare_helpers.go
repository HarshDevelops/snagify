package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/harshdevelops/snagify/internal/share"
)

// resolveBaselineURL fetches the /resolve endpoint on the share server to
// obtain the token-bearing download URL. The server returns {"url":"..."} when
// the pairing code (if any) is correct.
func resolveBaselineURL(baseURL, fingerprint, code string, timeout time.Duration) (string, error) {
	resolveURL := baseURL + "/resolve"
	if code != "" {
		resolveURL += "?code=" + code
	}

	data, err := share.Fetch(share.FetchOptions{
		URL:     resolveURL,
		Pin:     fingerprint,
		Timeout: timeout,
	})
	if err != nil {
		return "", err
	}

	var resp struct {
		URL string `json:"url"`
	}
	if err := json.Unmarshal(data, &resp); err != nil {
		return "", fmt.Errorf("parse resolve response: %w", err)
	}
	if resp.URL == "" {
		return "", fmt.Errorf("resolve returned empty URL")
	}
	return resp.URL, nil
}

// writeFile writes data to path, creating parent directories as needed.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
