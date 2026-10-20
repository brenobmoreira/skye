package files

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@t", "-c", "init.defaultBranch=main"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// repo has main with a.txt and docs/readme.md committed, then a branch "work" with one commit
// that changes a.txt.
func repo(t *testing.T) string {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	git(t, dir, "init", "-q")
	write(t, filepath.Join(dir, "a.txt"), "one\ntwo\n")
	write(t, filepath.Join(dir, "docs", "readme.md"), "# hi\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-qm", "base")
	git(t, dir, "checkout", "-qb", "work")
	write(t, filepath.Join(dir, "a.txt"), "one\ntwo\nthree\n")
	git(t, dir, "commit", "-qam", "three")
	return dir
}

func TestRootIsTheWorkTreeHoldingTheFolder(t *testing.T) {
	dir := repo(t)
	if root, ok := Root(filepath.Join(dir, "docs")); root != dir || !ok {
		t.Fatalf("root = %q %v", root, ok)
	}
	plain := t.TempDir()
	if root, ok := Root(plain); root != plain || ok {
		t.Fatalf("plain root = %q %v", root, ok)
	}
}

func TestChangesSinceHeadAndSinceBase(t *testing.T) {
	dir := repo(t)
	write(t, filepath.Join(dir, "docs", "readme.md"), "# hi\nmore\n")
	write(t, filepath.Join(dir, "new.env"), "A=1\nB=2")
	os.Remove(filepath.Join(dir, "a.txt"))

	work, err := ListChanges(dir, Work, nil)
	if err != nil {
		t.Fatal(err)
	}
	want := []Change{
		{Path: "a.txt", Status: "D", Removed: 3},
		{Path: "docs/readme.md", Status: "M", Added: 1},
		{Path: "new.env", Status: "?", Added: 2},
	}
	if !reflect.DeepEqual(work.Files, want) || work.Branch != "work" || work.Base != "HEAD" {
		t.Fatalf("work = %+v", work)
	}

	git(t, dir, "checkout", "-q", "a.txt")
	branch, err := ListChanges(dir, Branch, []string{"nope", "main"})
	if err != nil {
		t.Fatal(err)
	}
	want = []Change{
		{Path: "a.txt", Status: "M", Added: 1},
		{Path: "docs/readme.md", Status: "M", Added: 1},
		{Path: "new.env", Status: "?", Added: 2},
	}
	if !reflect.DeepEqual(branch.Files, want) || branch.Base != "main" {
		t.Fatalf("branch = %+v", branch)
	}
	if _, err := ListChanges(dir, Branch, []string{"nope"}); err == nil {
		t.Fatal("branch mode without a base must fail")
	}
}

func TestChangesOutsideGitFail(t *testing.T) {
	if _, err := ListChanges(t.TempDir(), Work, nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestDiffHasBothSides(t *testing.T) {
	dir := repo(t)
	write(t, filepath.Join(dir, "a.txt"), "one\n")
	d, err := DiffOf(dir, "a.txt", Branch, []string{"main"})
	if err != nil {
		t.Fatal(err)
	}
	if d.Old != "one\ntwo\n" || d.New != "one\n" || d.Path != "a.txt" {
		t.Fatalf("diff = %+v", d)
	}
	write(t, filepath.Join(dir, "n.txt"), "x\n")
	if d, _ := DiffOf(dir, "n.txt", Work, nil); d.Old != "" || d.New != "x\n" {
		t.Fatalf("new file diff = %+v", d)
	}
	os.Remove(filepath.Join(dir, "a.txt"))
	if d, _ := DiffOf(dir, "a.txt", Work, nil); d.Old != "one\ntwo\nthree\n" || d.New != "" {
		t.Fatalf("deleted file diff = %+v", d)
	}
}

func TestListShowsHiddenFilesButNotGit(t *testing.T) {
	dir := repo(t)
	write(t, filepath.Join(dir, ".env"), "A=1\n")
	entries, err := List(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name)
	}
	if strings.Join(names, ",") != "docs,.env,a.txt" || !entries[0].Dir {
		t.Fatalf("entries = %+v", entries)
	}
	if sub, _ := List(dir, "docs"); len(sub) != 1 || sub[0].Name != "readme.md" {
		t.Fatalf("docs = %+v", sub)
	}
}

func TestPathsStayInsideTheRoot(t *testing.T) {
	dir := repo(t)
	outside := t.TempDir()
	write(t, filepath.Join(outside, "secret"), "s")
	os.Symlink(outside, filepath.Join(dir, "link"))
	for _, rel := range []string{"../x", "/etc/passwd", "link/secret", "docs/../../x"} {
		if _, err := Read(dir, rel); err == nil {
			t.Errorf("read %q allowed", rel)
		}
	}
	if _, err := List(dir, "link"); err == nil {
		t.Error("list through symlink allowed")
	}
}

func TestReadAndWriteWithVersion(t *testing.T) {
	dir := repo(t)
	path := filepath.Join(dir, ".env")
	write(t, path, "A=1\n")
	os.Chmod(path, 0o600)
	f, err := Read(dir, ".env")
	if err != nil || f.Content != "A=1\n" || f.Version == "" || f.Binary {
		t.Fatalf("read = %+v %v", f, err)
	}
	saved, err := Write(dir, ".env", "A=2\n", f.Version)
	if err != nil {
		t.Fatal(err)
	}
	if data, _ := os.ReadFile(path); string(data) != "A=2\n" || saved.Version == f.Version {
		t.Fatalf("after write = %q %+v", data, saved)
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode())
	}
	if _, err := Write(dir, ".env", "A=3\n", f.Version); err == nil || !strings.Contains(err.Error(), "mudou") {
		t.Fatalf("stale write err = %v", err)
	}
	if _, err := Write(dir, "nope.txt", "x", ""); err == nil {
		t.Fatal("write created a file")
	}
	if _, err := Write(dir, ".git/config", "x", ""); err == nil {
		t.Fatal("write inside .git allowed")
	}
	os.Symlink(".git/config", filepath.Join(dir, "cfg"))
	g, _ := Read(dir, "cfg")
	if _, err := Write(dir, "cfg", "x", g.Version); err == nil || !strings.Contains(err.Error(), ".git") {
		t.Fatalf("write through link into .git: %v", err)
	}
}

func TestWriteThroughSymlinkKeepsTheLink(t *testing.T) {
	dir := repo(t)
	write(t, filepath.Join(dir, "real.env"), "A=1\n")
	os.Symlink("real.env", filepath.Join(dir, ".env"))
	f, _ := Read(dir, ".env")
	if _, err := Write(dir, ".env", "A=2\n", f.Version); err != nil {
		t.Fatal(err)
	}
	if target, err := os.Readlink(filepath.Join(dir, ".env")); err != nil || target != "real.env" {
		t.Fatalf("link = %q %v", target, err)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, "real.env")); string(data) != "A=2\n" {
		t.Fatalf("real = %q", data)
	}
}

func TestReadFlagsBinaryAndLargeFiles(t *testing.T) {
	dir := repo(t)
	write(t, filepath.Join(dir, "bin"), "a\x00b")
	if f, _ := Read(dir, "bin"); !f.Binary || f.Content != "" {
		t.Fatalf("binary = %+v", f)
	}
	write(t, filepath.Join(dir, "big"), strings.Repeat("x", MaxSize+1))
	if f, _ := Read(dir, "big"); !f.TooBig || f.Content != "" {
		t.Fatalf("big = %+v", f.TooBig)
	}
}

func TestEditorCommand(t *testing.T) {
	dir := repo(t)
	if args, err := EditorArgs(dir, ""); err != nil || !reflect.DeepEqual(args, []string{dir}) {
		t.Fatalf("folder = %v %v", args, err)
	}
	if args, err := EditorArgs(dir, "docs/readme.md"); err != nil || !reflect.DeepEqual(args, []string{dir, "-g", filepath.Join(dir, "docs", "readme.md")}) {
		t.Fatalf("file = %v %v", args, err)
	}
	if _, err := EditorArgs(dir, "../x"); err == nil {
		t.Fatal("outside path allowed")
	}
}
