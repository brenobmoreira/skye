package update

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
)

func TestInstallReplacesTheBinaryWhole(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "new"), filepath.Join(dir, "bin", "skye")
	os.MkdirAll(filepath.Dir(dst), 0o755)
	os.WriteFile(dst, []byte("old"), 0o755)
	os.WriteFile(src, []byte("new"), 0o644)
	if err := Install(src, dst); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(dst)
	info, _ := os.Stat(dst)
	if string(data) != "new" || info.Mode().Perm() != 0o755 {
		t.Fatalf("content %q mode %v", data, info.Mode())
	}
	if left, _ := os.ReadDir(filepath.Dir(dst)); len(left) != 1 {
		t.Fatalf("leftovers %v", left)
	}
}

func TestRunBuildsInTheRepoThenInstalls(t *testing.T) {
	dir := t.TempDir()
	repo, target := filepath.Join(dir, "repo"), filepath.Join(dir, "skye")
	os.MkdirAll(repo, 0o755)
	var ran []string
	err := Run(Options{Repo: repo, Target: target, Run: func(d, name string, args ...string) ([]byte, error) {
		ran = append(ran, d+": "+name+" "+strings.Join(args, " "))
		return nil, os.WriteFile(filepath.Join(repo, "skye"), []byte("built"), 0o755)
	}})
	if err != nil {
		t.Fatal(err)
	}
	if len(ran) != 1 || ran[0] != repo+": make build" {
		t.Fatalf("ran %q", ran)
	}
	if data, _ := os.ReadFile(target); string(data) != "built" {
		t.Fatalf("target %q", data)
	}
}

func TestRunStopsWhenTheBuildFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "skye")
	os.WriteFile(target, []byte("old"), 0o755)
	err := Run(Options{Repo: dir, Target: target, Run: func(string, string, ...string) ([]byte, error) {
		return []byte("lots of output\nsrc/App.tsx(3,1): error TS1005"), errors.New("exit status 2")
	}})
	if err == nil || !strings.Contains(err.Error(), "error TS1005") {
		t.Fatalf("err = %v", err)
	}
	if data, _ := os.ReadFile(target); string(data) != "old" {
		t.Fatal("binary replaced after a failed build")
	}
	if err := Run(Options{Target: target}); err == nil {
		t.Fatal("empty repo accepted")
	}
}

func TestVersionFollowsTheIndexPage(t *testing.T) {
	a := fstest.MapFS{"index.html": {Data: []byte(`<script src="/assets/index-abc.js">`)}}
	b := fstest.MapFS{"index.html": {Data: []byte(`<script src="/assets/index-def.js">`)}}
	if Version(a) == "" || Version(a) != Version(a) || Version(a) == Version(b) {
		t.Fatalf("a %q b %q", Version(a), Version(b))
	}
	if Version(fstest.MapFS{}) != "" {
		t.Fatal("version without an index page")
	}
}

func TestShellPathTakesWhatFollowsTheLastMark(t *testing.T) {
	for out, want := range map[string]string{
		pathMark + "/a:/b":                       "/a:/b",
		"bem-vindo\n" + pathMark + "/nvm/bin:/b": "/nvm/bin:/b",
		pathMark + "x" + pathMark + "/c\n":       "/c",
		"sem marca":                              "",
	} {
		if got := parsePath(out); got != want {
			t.Errorf("parsePath(%q) = %q, want %q", out, got, want)
		}
	}
}

func TestWithPathReplacesOrAddsPATH(t *testing.T) {
	got := withPath([]string{"HOME=/h", "PATH=/usr/bin", "X=1"}, "/nvm/bin:/usr/bin")
	if strings.Join(got, " ") != "HOME=/h PATH=/nvm/bin:/usr/bin X=1" {
		t.Fatalf("replaced %q", got)
	}
	if got := withPath([]string{"HOME=/h"}, "/a"); strings.Join(got, " ") != "HOME=/h PATH=/a" {
		t.Fatalf("added %q", got)
	}
	if got := withPath([]string{"PATH=/usr/bin"}, ""); strings.Join(got, " ") != "PATH=/usr/bin" {
		t.Fatalf("empty path changed the env: %q", got)
	}
}
