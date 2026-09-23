package launch

import (
	"os/exec"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestArgv(t *testing.T) {
	if got := Argv("/bin/bash", "  "); !reflect.DeepEqual(got, []string{"/bin/bash", "-li"}) {
		t.Fatalf("empty command: %q", got)
	}
	got := Argv("/bin/bash", "cd ~/projects/blog && claude")
	want := []string{"/bin/bash", "-lic", "cd ~/projects/blog && claude\nexec '/bin/bash' -li"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestShellQuote(t *testing.T) {
	if got := ShellQuote("it's"); got != `'it'\''s'` {
		t.Fatalf("got %s", got)
	}
}

func TestCommandWithCommentOrBackgroundStillFallsBackToShell(t *testing.T) {
	for _, command := range []string{"echo first # a comment", "echo first &"} {
		argv := Argv("/bin/sh", command)
		cmd := exec.Command(argv[0], argv[1:]...)
		cmd.Stdin = strings.NewReader("echo second\n")
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("%q: %v: %s", command, err, out)
		}
		if !strings.Contains(string(out), "first") || !strings.Contains(string(out), "second") {
			t.Fatalf("%q: shell did not continue after the command: %q", command, out)
		}
	}
}

func TestWriteReadRemoveList(t *testing.T) {
	dir := t.TempDir()
	spec := Spec{Name: "blog", Preset: "blog", Cwd: "/home/demo", Command: "claude"}
	if err := Write(dir, "a1", spec); err != nil {
		t.Fatal(err)
	}
	if err := Write(dir, "b2", Spec{Name: "shell"}); err != nil {
		t.Fatal(err)
	}
	got, err := Read(dir, "a1")
	if err != nil || got != spec {
		t.Fatalf("Read = %+v %v", got, err)
	}
	ids, err := List(dir)
	sort.Strings(ids)
	if err != nil || !reflect.DeepEqual(ids, []string{"a1", "b2"}) {
		t.Fatalf("List = %v %v", ids, err)
	}
	if err := Remove(dir, "a1"); err != nil {
		t.Fatal(err)
	}
	if err := Remove(dir, "a1"); err != nil {
		t.Fatalf("second Remove: %v", err)
	}
	if _, err := Read(dir, "a1"); err == nil {
		t.Fatal("Read after Remove should fail")
	}
	if ids, err := List(t.TempDir() + "/missing"); err != nil || len(ids) != 0 {
		t.Fatalf("List missing dir = %v %v", ids, err)
	}
}
