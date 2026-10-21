package app

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
)

func gitRepo(t *testing.T) string {
	t.Helper()
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	for _, args := range [][]string{
		{"init", "-q", "-b", "trunk"},
		{"commit", "-q", "--allow-empty", "-m", "base"},
		{"checkout", "-qb", "work"},
	} {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t"}, args...)...)
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	return dir
}

func TestFilesFollowTheTerminalFolder(t *testing.T) {
	dir := gitRepo(t)
	os.WriteFile(filepath.Join(dir, ".env"), []byte("A=1\n"), 0o600)
	cfg := config.Default()
	cfg.Repos = []config.Repo{{Name: "demo", Path: dir, Base: "trunk"}}
	h := newHarness(t, cfg)
	term, _ := h.app.NewTerminal("")
	h.hook(term.ID, "SessionStart", func(e *hooks.Event) { e.Cwd = dir })

	ch, err := h.app.Changes(term.ID, "branch")
	if err != nil {
		t.Fatal(err)
	}
	if ch.Root != dir || ch.Base != "trunk" || len(ch.Files) != 1 || ch.Files[0].Path != ".env" {
		t.Fatalf("changes = %+v", ch)
	}
	d, err := h.app.Diff(term.ID, ".env", "work")
	if err != nil || d.New != "A=1\n" {
		t.Fatalf("diff = %+v %v", d, err)
	}
	f, err := h.app.ReadFile(term.ID, ".env")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.app.WriteFile(term.ID, ".env", "A=2\n", f.Version); err != nil {
		t.Fatal(err)
	}
	if entries, err := h.app.ListDir(term.ID, ""); err != nil || len(entries) != 1 {
		t.Fatalf("entries = %+v %v", entries, err)
	}
	if _, err := h.app.Changes(term.ID, "nope"); err == nil {
		t.Fatal("unknown mode accepted")
	}
	if _, err := h.app.ReadFile("t99", ".env"); err == nil {
		t.Fatal("unknown terminal accepted")
	}
}

func TestOpenInEditorUsesTheTerminalRoot(t *testing.T) {
	dir := gitRepo(t)
	h := newHarness(t, config.Default())
	var got []string
	h.app.o.OpenEditor = func(root, rel string) error { got = []string{root, rel}; return nil }
	term, _ := h.app.NewTerminal("")
	h.hook(term.ID, "SessionStart", func(e *hooks.Event) { e.Cwd = filepath.Join(dir) })
	if err := h.app.OpenInEditor(term.ID, "a.txt"); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, []string{dir, "a.txt"}) {
		t.Fatalf("opened %v", got)
	}
}
