// Package files lets the panel next to a terminal show what changed in its folder, read a file and
// save a short edit. Every path is relative to the terminal's root and must stay inside it.
package files

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// MaxSize is the largest file the panel reads or diffs.
const MaxSize = 2 << 20

// emptyTree is git's hash for a tree with nothing in it: the base of a repo with no commits yet.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

type Mode string

const (
	// Work compares the working tree with the last commit: what is not committed yet.
	Work Mode = "work"
	// Branch compares the working tree with where the branch left its base.
	Branch Mode = "branch"
)

// Root is the folder a terminal's files are shown from: the git work tree holding cwd, or cwd.
func Root(cwd string) (string, bool) {
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return cwd, false
	}
	return strings.TrimSpace(string(out)), true
}

// CommonDir is the .git folder a work tree shares with its other worktrees.
func CommonDir(dir string) (string, bool) {
	out, err := exec.Command("git", "-C", dir, "rev-parse", "--path-format=absolute", "--git-common-dir").Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func run(root string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", root, "-c", "core.quotePath=false"}, args...)...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git %s: %s", args[0], strings.TrimSpace(stderr.String()))
	}
	return out, nil
}

func exists(root, rev string) bool {
	return exec.Command("git", "-C", root, "rev-parse", "--verify", "--quiet", rev+"^{commit}").Run() == nil
}

type Change struct {
	Path    string `json:"path"`
	Status  string `json:"status"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Binary  bool   `json:"binary,omitempty"`
}

type Changes struct {
	Root   string   `json:"root"`
	Branch string   `json:"branch"`
	Base   string   `json:"base"`
	Files  []Change `json:"files"`
}

// since returns the commit a mode compares with and the name to show for it. Branch mode takes the
// first of bases that exists and compares with where HEAD left it.
func since(root string, mode Mode, bases []string) (rev, name string, err error) {
	head := "HEAD"
	if !exists(root, head) {
		head = emptyTree
	}
	if mode != Branch {
		return head, "HEAD", nil
	}
	for _, b := range bases {
		if b == "" || strings.HasPrefix(b, "-") || !exists(root, b) {
			continue
		}
		out, err := run(root, "merge-base", b, "HEAD")
		if err != nil {
			return "", "", err
		}
		return strings.TrimSpace(string(out)), b, nil
	}
	return "", "", errors.New("não achei a branch base deste repo")
}

// ListChanges lists the files that differ from the commit the mode compares with, untracked
// ones included, sorted by path.
func ListChanges(root string, mode Mode, bases []string) (Changes, error) {
	rev, name, err := since(root, mode, bases)
	if err != nil {
		return Changes{}, err
	}
	branch := ""
	if out, err := run(root, "branch", "--show-current"); err == nil {
		branch = strings.TrimSpace(string(out))
	}
	status, err := run(root, "diff", "--no-renames", "--name-status", "-z", rev)
	if err != nil {
		return Changes{}, err
	}
	numstat, err := run(root, "diff", "--no-renames", "--numstat", "-z", rev)
	if err != nil {
		return Changes{}, err
	}
	untracked, err := run(root, "ls-files", "--others", "--exclude-standard", "-z")
	if err != nil {
		return Changes{}, err
	}
	byPath := map[string]*Change{}
	var list []Change
	fields := strings.Split(strings.TrimSuffix(string(status), "\x00"), "\x00")
	for i := 0; i+1 < len(fields); i += 2 {
		if fields[i] != "" {
			list = append(list, Change{Path: fields[i+1], Status: fields[i][:1]})
		}
	}
	for _, line := range strings.Split(strings.TrimSuffix(string(numstat), "\x00"), "\x00") {
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) != 3 {
			continue
		}
		c := Change{Path: parts[2]}
		if parts[0] == "-" {
			c.Binary = true
		} else {
			c.Added, _ = strconv.Atoi(parts[0])
			c.Removed, _ = strconv.Atoi(parts[1])
		}
		byPath[c.Path] = &c
	}
	for i, c := range list {
		if n, ok := byPath[c.Path]; ok {
			list[i].Added, list[i].Removed, list[i].Binary = n.Added, n.Removed, n.Binary
		}
	}
	for _, p := range strings.Split(strings.TrimSuffix(string(untracked), "\x00"), "\x00") {
		if p == "" {
			continue
		}
		c := Change{Path: p, Status: "?"}
		if data, err := readLimited(filepath.Join(root, p)); err == nil && !binary(data) {
			c.Added = lines(data)
		} else if err == nil {
			c.Binary = true
		}
		list = append(list, c)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Path < list[j].Path })
	if list == nil {
		list = []Change{}
	}
	return Changes{Root: root, Branch: branch, Base: name, Files: list}, nil
}

type Diff struct {
	Path   string `json:"path"`
	Old    string `json:"old"`
	New    string `json:"new"`
	Binary bool   `json:"binary,omitempty"`
	TooBig bool   `json:"tooBig,omitempty"`
}

// DiffOf returns a file as it was in the commit the mode compares with and as it is on disk; a
// side where the file does not exist is empty.
func DiffOf(root, rel string, mode Mode, bases []string) (Diff, error) {
	full, err := resolve(root, rel, true)
	if err != nil {
		return Diff{}, err
	}
	rel, _ = filepath.Rel(root, full)
	rev, _, err := since(root, mode, bases)
	if err != nil {
		return Diff{}, err
	}
	d := Diff{Path: filepath.ToSlash(rel)}
	old, _ := run(root, "show", rev+":"+d.Path)
	cur, err := readLimited(full)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		if errors.Is(err, errTooBig) {
			d.TooBig = true
			return d, nil
		}
		return Diff{}, err
	}
	if len(old) > MaxSize {
		d.TooBig = true
		return d, nil
	}
	if binary(old) || binary(cur) {
		d.Binary = true
		return d, nil
	}
	d.Old, d.New = string(old), string(cur)
	return d, nil
}

type Entry struct {
	Name string `json:"name"`
	Dir  bool   `json:"dir"`
	Size int64  `json:"size"`
}

// List returns a folder's entries, folders first, without .git.
func List(root, rel string) ([]Entry, error) {
	full, err := resolve(root, rel, false)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(full)
	if err != nil {
		return nil, err
	}
	list := []Entry{}
	for _, it := range items {
		if it.Name() == ".git" {
			continue
		}
		e := Entry{Name: it.Name(), Dir: it.IsDir()}
		if info, err := os.Stat(filepath.Join(full, it.Name())); err == nil {
			e.Dir = info.IsDir()
			if !e.Dir {
				e.Size = info.Size()
			}
		}
		list = append(list, e)
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].Dir != list[j].Dir {
			return list[i].Dir
		}
		return strings.ToLower(list[i].Name) < strings.ToLower(list[j].Name)
	})
	return list, nil
}

type File struct {
	Path    string `json:"path"`
	Content string `json:"content"`
	// Version identifies the content read, so a save does not overwrite a newer change.
	Version string `json:"version"`
	Size    int64  `json:"size"`
	Binary  bool   `json:"binary,omitempty"`
	TooBig  bool   `json:"tooBig,omitempty"`
}

func Read(root, rel string) (File, error) {
	full, err := resolve(root, rel, false)
	if err != nil {
		return File{}, err
	}
	info, err := os.Stat(full)
	if err != nil {
		return File{}, err
	}
	if !info.Mode().IsRegular() {
		return File{}, fmt.Errorf("%s não é um arquivo", rel)
	}
	f := File{Path: filepath.ToSlash(rel), Size: info.Size()}
	data, err := readLimited(full)
	if errors.Is(err, errTooBig) {
		f.TooBig = true
		return f, nil
	}
	if err != nil {
		return File{}, err
	}
	f.Version = version(data)
	if binary(data) {
		f.Binary = true
		return f, nil
	}
	f.Content = string(data)
	return f, nil
}

// Write replaces the content of an existing file, as long as it still holds the version the
// editor read. It writes through a symlink to its target and keeps the file's permissions.
func Write(root, rel, content, expected string) (File, error) {
	full, err := resolve(root, rel, false)
	if err != nil {
		return File{}, err
	}
	realRoot, _ := filepath.EvalSymlinks(root)
	inside, _ := filepath.Rel(realRoot, full)
	for _, part := range strings.Split(filepath.ToSlash(inside), "/") {
		if part == ".git" {
			return File{}, errors.New("não edito arquivos dentro do .git")
		}
	}
	info, err := os.Stat(full)
	if err != nil {
		return File{}, err
	}
	if !info.Mode().IsRegular() {
		return File{}, fmt.Errorf("%s não é um arquivo", rel)
	}
	current, err := readLimited(full)
	if err != nil {
		return File{}, err
	}
	if version(current) != expected {
		return File{}, errors.New("o arquivo mudou no disco desde que foi aberto; recarregue antes de salvar")
	}
	tmp, err := os.CreateTemp(filepath.Dir(full), "."+filepath.Base(full)+".skye-*")
	if err != nil {
		return File{}, err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return File{}, err
	}
	if err := tmp.Chmod(info.Mode().Perm()); err != nil {
		tmp.Close()
		return File{}, err
	}
	if err := tmp.Close(); err != nil {
		return File{}, err
	}
	if err := os.Rename(tmp.Name(), full); err != nil {
		return File{}, err
	}
	return Read(root, rel)
}

// EditorArgs are the arguments for `code` to open the root, and the file in it when rel is set.
func EditorArgs(root, rel string) ([]string, error) {
	if rel == "" {
		return []string{root}, nil
	}
	full, err := resolve(root, rel, true)
	if err != nil {
		return nil, err
	}
	return []string{root, "-g", full}, nil
}

// OpenInEditor starts VS Code without waiting for it.
func OpenInEditor(root, rel string) error {
	args, err := EditorArgs(root, rel)
	if err != nil {
		return err
	}
	code, err := exec.LookPath("code")
	if err != nil {
		return errors.New("não achei o comando code do VS Code no PATH")
	}
	cmd := exec.Command(code, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}

// resolve turns a path relative to root into an absolute one that, symlinks followed, is still
// inside root. With missing set, a path that does not exist yet (a deleted file) is allowed.
func resolve(root, rel string, missing bool) (string, error) {
	clean := filepath.Clean(filepath.FromSlash(rel))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("caminho fora da pasta do terminal: %q", rel)
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	full := filepath.Join(realRoot, clean)
	real, err := filepath.EvalSymlinks(full)
	if errors.Is(err, fs.ErrNotExist) && missing {
		real, err = filepath.EvalSymlinks(filepath.Dir(full))
		if err == nil {
			real = filepath.Join(real, filepath.Base(full))
		}
	}
	if err != nil {
		return "", err
	}
	if real != realRoot && !strings.HasPrefix(real, realRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("caminho fora da pasta do terminal: %q", rel)
	}
	return real, nil
}

var errTooBig = errors.New("arquivo grande demais")

func readLimited(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	data := make([]byte, MaxSize+1)
	n, err := f.Read(data)
	for err == nil && n < len(data) {
		var m int
		m, err = f.Read(data[n:])
		n += m
	}
	if n > MaxSize {
		return nil, errTooBig
	}
	return data[:n], nil
}

// lines counts like git does: a last line without a newline still counts.
func lines(data []byte) int {
	n := bytes.Count(data, []byte("\n"))
	if len(data) > 0 && data[len(data)-1] != '\n' {
		n++
	}
	return n
}

func binary(data []byte) bool {
	return bytes.IndexByte(data[:min(len(data), 8000)], 0) >= 0
}

func version(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
