package capture

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject_WalksUp(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}

	info := detectProject(nested, "")
	if info.Root != root {
		t.Errorf("Root = %q, want %q", info.Root, root)
	}
	if len(info.Manifests) != 1 || info.Manifests[0] != "go.mod" {
		t.Errorf("Manifests = %v, want [go.mod]", info.Manifests)
	}
}

func TestDetectProject_NameFromPackageJSON(t *testing.T) {
	root := t.TempDir()
	pkg := `{"name": "checkout-service", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(pkg), 0o644); err != nil {
		t.Fatal(err)
	}

	info := detectProject(root, "")
	if info.Name != "checkout-service" {
		t.Errorf("Name = %q, want checkout-service", info.Name)
	}
}

func TestDetectProject_Override(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "Cargo.toml"), []byte("[package]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info := detectProject("/somewhere/else", root)
	if info.Root != root {
		t.Errorf("Root = %q, want %q", info.Root, root)
	}
}

func TestDetectProject_NoManifest(t *testing.T) {
	dir := t.TempDir()
	info := detectProject(dir, "")
	if info.Root != dir {
		t.Errorf("Root = %q, want %q", info.Root, dir)
	}
	if len(info.Manifests) != 0 {
		t.Errorf("expected no manifests, got %v", info.Manifests)
	}
}
