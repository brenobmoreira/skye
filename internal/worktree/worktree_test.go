package worktree

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t", "GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v: %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func repo(t *testing.T) (string, string) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	r := filepath.Join(root, "app")
	if err := os.MkdirAll(r, 0o755); err != nil {
		t.Fatal(err)
	}
	git(t, r, "init", "-q", "-b", "develop")
	git(t, r, "commit", "-q", "--allow-empty", "-m", "first")
	return r, root
}

func TestCreateAddsABranchInItsOwnFolder(t *testing.T) {
	r, root := repo(t)
	path, err := Create(r, "develop", root, "feature/login")
	if err != nil {
		t.Fatal(err)
	}
	if path != filepath.Join(root, "feature-login") {
		t.Fatalf("path = %s", path)
	}
	if got := git(t, path, "rev-parse", "--abbrev-ref", "HEAD"); got != "feature/login" {
		t.Fatalf("branch = %s", got)
	}
}

func TestCreateRefusesBadNamesTakenFoldersAndMissingBases(t *testing.T) {
	r, root := repo(t)
	if _, err := Create(r, "develop", root, "bad..name"); err == nil || !strings.Contains(err.Error(), "nome de branch inválido") {
		t.Fatalf("bad name: %v", err)
	}
	if _, err := Create(r, "develop", root, "-x"); err == nil {
		t.Fatal("name starting with a dash accepted")
	}
	if err := os.MkdirAll(filepath.Join(root, "taken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Create(r, "develop", root, "taken"); err == nil || !strings.Contains(err.Error(), "já existe") {
		t.Fatalf("taken folder: %v", err)
	}
	if _, err := Create(r, "nope", root, "other"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("missing base: %v", err)
	}
}
