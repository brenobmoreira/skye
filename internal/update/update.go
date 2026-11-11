// Package update rebuilds skye from its repository and puts the new binary in place of the one
// running. The running server then restarts itself on the new binary; the terminals live in tmux
// and do not notice.
package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Options struct {
	// Repo is the checkout skye was built from; Target the installed binary to replace.
	Repo   string
	Target string
	// Run runs a command in a folder and returns its output (exec outside tests).
	Run func(dir, name string, args ...string) ([]byte, error)
}

// tailLines is how much of a failed build's output the error keeps.
const tailLines = 15

func Exec(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	return cmd.CombinedOutput()
}

func Run(o Options) error {
	if o.Repo == "" {
		return errors.New("não sei onde está o repositório da skye; compile com make build")
	}
	if o.Run == nil {
		o.Run = Exec
	}
	if out, err := o.Run(o.Repo, "make", "build"); err != nil {
		return fmt.Errorf("a compilação falhou (%v):\n%s", err, tail(string(out)))
	}
	return Install(filepath.Join(o.Repo, "skye"), o.Target)
}

func tail(out string) string {
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) > tailLines {
		lines = lines[len(lines)-tailLines:]
	}
	return strings.Join(lines, "\n")
}

// Install copies src over dst through a temporary file in dst's folder: a running binary cannot
// be written in place, but it can be replaced by a rename.
func Install(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+".*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	fail := func(err error) error {
		tmp.Close()
		os.Remove(name)
		return err
	}
	if _, err := io.Copy(tmp, in); err != nil {
		return fail(err)
	}
	if err := tmp.Chmod(0o755); err != nil {
		return fail(err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(name)
		return err
	}
	if err := os.Rename(name, dst); err != nil {
		os.Remove(name)
		return err
	}
	return nil
}

// Version names the interface build: the index page lists the hashed bundles, so it changes
// whenever the interface does. Empty when there is no index page.
func Version(assets fs.FS) string {
	data, err := fs.ReadFile(assets, "index.html")
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:6])
}
