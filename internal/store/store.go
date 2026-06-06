// Package store handles reading and writing snapshot JSON files.
package store

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/HarshDevelops/snagify/internal/model"
)

// Save writes a snapshot to path as indented JSON.
func Save(path string, s model.Snapshot) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// Load reads and parses a snapshot file, returning a clear error for malformed
// or missing files.
func Load(path string) (model.Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return model.Snapshot{}, fmt.Errorf("read %s: %w", path, err)
	}
	var s model.Snapshot
	if err := json.Unmarshal(data, &s); err != nil {
		return model.Snapshot{}, fmt.Errorf("parse %s: not a valid snapshot (%w)", path, err)
	}
	return s, nil
}
