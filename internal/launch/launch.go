package launch

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

type Spec struct {
	Name    string `json:"name"`
	Preset  string `json:"preset"`
	Cwd     string `json:"cwd"`
	Command string `json:"command"`
	Order   int    `json:"order,omitempty"`
}

func file(dir, id string) string {
	return filepath.Join(dir, id+".json")
}

func Write(dir, id string, s Spec) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := file(dir, id) + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, file(dir, id))
}

func Read(dir, id string) (Spec, error) {
	data, err := os.ReadFile(file(dir, id))
	if err != nil {
		return Spec{}, err
	}
	var s Spec
	if err := json.Unmarshal(data, &s); err != nil {
		return Spec{}, fmt.Errorf("%s: %w", file(dir, id), err)
	}
	return s, nil
}

func Remove(dir, id string) error {
	err := os.Remove(file(dir, id))
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for _, e := range entries {
		if name, ok := strings.CutSuffix(e.Name(), ".json"); ok && !e.IsDir() {
			ids = append(ids, name)
		}
	}
	return ids, nil
}

func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func Argv(shell, command string) []string {
	if strings.TrimSpace(command) == "" {
		return []string{shell, "-li"}
	}
	return []string{shell, "-lic", command + "\nexec " + ShellQuote(shell) + " -li"}
}

func Exec(dir, id, shell string) error {
	spec, err := Read(dir, id)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skye: %v — abrindo só o shell\n", err)
		spec = Spec{}
	}
	cwd := spec.Cwd
	if cwd == "" {
		cwd, _ = os.UserHomeDir()
	}
	if err := os.Chdir(cwd); err != nil {
		fmt.Fprintf(os.Stderr, "skye: %v\n", err)
	}
	argv := Argv(shell, spec.Command)
	bin, err := exec.LookPath(argv[0])
	if err != nil {
		return err
	}
	return syscall.Exec(bin, argv, os.Environ())
}
