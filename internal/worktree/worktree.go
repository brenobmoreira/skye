// Package worktree creates git worktrees for new terminals. It never removes one: a worktree can
// hold work that is not committed anywhere else.
package worktree

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Create adds a worktree of repo on a new branch cut from base, in dir/<branch> with slashes
// turned into dashes, and returns its path.
func Create(repo, base, dir, branch string) (string, error) {
	if strings.HasPrefix(branch, "-") || exec.Command("git", "check-ref-format", "--branch", branch).Run() != nil {
		return "", fmt.Errorf("nome de branch inválido: %q", branch)
	}
	path := filepath.Join(dir, strings.ReplaceAll(branch, "/", "-"))
	if _, err := os.Stat(path); err == nil {
		return "", fmt.Errorf("a pasta %s já existe", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	out, err := exec.Command("git", "-C", repo, "worktree", "add", "-b", branch, path, base).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git worktree add (base %s): %s", base, strings.TrimSpace(string(out)))
	}
	return path, nil
}
