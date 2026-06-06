package capture

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func gitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=t@e.x",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=t@e.x",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestCaptureGit_NotARepo(t *testing.T) {
	g := captureGit(t.TempDir())
	if g.Present {
		t.Error("expected Present=false for non-repo")
	}
}

func TestCaptureGit_BranchCommitDirtyUntracked(t *testing.T) {
	if !gitAvailable() {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "checkout", "-q", "-b", "main")

	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f.txt")
	runGit(t, dir, "commit", "-q", "-m", "init")

	g := captureGit(dir)
	if !g.Present {
		t.Fatal("expected Present=true")
	}
	if g.Branch != "main" {
		t.Errorf("branch = %q, want main", g.Branch)
	}
	if g.Commit == "" {
		t.Error("expected a commit hash")
	}
	if g.Dirty || g.Untracked {
		t.Errorf("clean repo flagged dirty=%v untracked=%v", g.Dirty, g.Untracked)
	}

	// Modify tracked file -> dirty.
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Add an untracked file -> untracked.
	if err := os.WriteFile(filepath.Join(dir, "new.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	g2 := captureGit(dir)
	if !g2.Dirty {
		t.Error("expected Dirty=true after modifying tracked file")
	}
	if !g2.Untracked {
		t.Error("expected Untracked=true after adding untracked file")
	}
}
