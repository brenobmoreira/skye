package places

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddListRemoveAndReopen(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, "projects", "ai_livia_copilot"), 0o755); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "skye", "paths.json")
	s, err := Open(file, home)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Add(" LivIA ", " ~/projects/ai_livia_copilot "); err != nil {
		t.Fatal(err)
	}
	if err := s.Add("home", home); err != nil {
		t.Fatal(err)
	}
	want := []Place{{Name: "LivIA", Path: "~/projects/ai_livia_copilot"}, {Name: "home", Path: home}}
	if got := s.List(); len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("list = %+v", got)
	}
	again, err := Open(file, home)
	if err != nil || len(again.List()) != 2 {
		t.Fatalf("reopen = %+v %v", again.List(), err)
	}
	p, ok := again.Get("livia")
	if !ok || s.Expand(p.Path) != filepath.Join(home, "projects", "ai_livia_copilot") {
		t.Fatalf("get = %+v %v", p, ok)
	}
	if err := again.Remove("LIVIA"); err != nil {
		t.Fatal(err)
	}
	if got := again.List(); len(got) != 1 || got[0].Name != "home" {
		t.Fatalf("after remove = %+v", got)
	}
}

func TestAddRefusesBadEntries(t *testing.T) {
	home := t.TempDir()
	if err := os.WriteFile(filepath.Join(home, "a-file"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	s, _ := Open(filepath.Join(home, "p.json"), home)
	if err := s.Add("x", home); err != nil {
		t.Fatal(err)
	}
	cases := map[string][2]string{
		"sem nome":        {" ", home},
		"sem path":        {"y", " "},
		"já existe":       {"X", home},
		"não existe":      {"y", "~/nope"},
		"não é uma pasta": {"y", "~/a-file"},
	}
	for want, c := range cases {
		if err := s.Add(c[0], c[1]); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("Add(%q, %q) = %v, want %q", c[0], c[1], err, want)
		}
	}
	if err := s.Remove("nobody"); err == nil {
		t.Error("removed an unknown place")
	}
}

func TestOpenMissingOrBrokenFile(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(filepath.Join(dir, "none.json"), dir)
	if err != nil || len(s.List()) != 0 {
		t.Fatalf("missing = %+v %v", s.List(), err)
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{"), 0o600)
	if s, err := Open(bad, dir); err == nil || len(s.List()) != 0 {
		t.Fatalf("broken = %+v %v", s.List(), err)
	}
}
