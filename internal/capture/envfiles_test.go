package capture

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func TestReadEnvKeys(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "# comment\nDATABASE_URL=\nexport JWT_SECRET=changeme\n\nPORT=3000\nINVALID\n")

	keys, ok := readEnvKeys(filepath.Join(dir, ".env.example"))
	if !ok {
		t.Fatal("expected file to exist")
	}
	want := []string{"DATABASE_URL", "JWT_SECRET", "PORT"}
	sort.Strings(keys)
	sort.Strings(want)
	if !reflect.DeepEqual(keys, want) {
		t.Errorf("keys = %v, want %v", keys, want)
	}
}

func TestReadEnvKeysMissingFile(t *testing.T) {
	_, ok := readEnvKeys(filepath.Join(t.TempDir(), "nope"))
	if ok {
		t.Error("expected ok=false for missing file")
	}
}

func TestCaptureEnvFiles_MissingKeys(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DATABASE_URL=\nJWT_SECRET=\nPORT=\n")
	writeFile(t, dir, ".env", "PORT=3000\n")

	got := captureEnvFiles(dir, false)
	if !got.EnvExampleExists || !got.EnvExists {
		t.Fatalf("expected both files to exist, got %+v", got)
	}
	want := []string{"DATABASE_URL", "JWT_SECRET"}
	sort.Strings(got.MissingKeys)
	if !reflect.DeepEqual(got.MissingKeys, want) {
		t.Errorf("missing = %v, want %v", got.MissingKeys, want)
	}
}

func TestCaptureEnvFiles_NoEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "A=\nB=\n")

	got := captureEnvFiles(dir, false)
	if got.EnvExists {
		t.Error("expected EnvExists=false")
	}
	if len(got.MissingKeys) != 2 {
		t.Errorf("expected all example keys missing, got %v", got.MissingKeys)
	}
}
