package resume

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func conv(id string, minute int) Conversation {
	return Conversation{SessionID: id, Cwd: "/home/demo/blog", Title: "t " + id, EndedAt: time.Date(2026, 9, 23, 10, minute, 0, 0, time.UTC)}
}

func TestAddListPersist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state", "conversations.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Add(conv("a", 1)); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(conv("b", 2)); err != nil {
		t.Fatal(err)
	}
	if err := s.Add(conv("a", 3)); err != nil {
		t.Fatal(err)
	}
	list := s.List()
	if len(list) != 2 || list[0].SessionID != "a" || list[1].SessionID != "b" {
		t.Fatalf("list = %+v", list)
	}
	reopened, err := Open(path)
	if err != nil || len(reopened.List()) != 2 {
		t.Fatalf("reopen: %v %+v", err, reopened.List())
	}
	if c, ok := reopened.Get("b"); !ok || c.Title != "t b" {
		t.Fatalf("Get = %+v %v", c, ok)
	}
	if err := reopened.Remove("a"); err != nil {
		t.Fatal(err)
	}
	if _, ok := reopened.Get("a"); ok {
		t.Fatal("a still present")
	}
}

func TestLimit(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "c.json"))
	for i := 0; i < Limit+5; i++ {
		if err := s.Add(conv(fmt.Sprintf("id-%d", i), i%60)); err != nil {
			t.Fatal(err)
		}
	}
	list := s.List()
	if len(list) != Limit || list[0].SessionID != fmt.Sprintf("id-%d", Limit+4) {
		t.Fatalf("len=%d first=%s", len(list), list[0].SessionID)
	}
}

func TestRejectsUnsafeSessionIDs(t *testing.T) {
	s, _ := Open(filepath.Join(t.TempDir(), "c.json"))
	for _, bad := range []string{"", "abc; rm -rf ~", "a b", "$(x)", "id\n"} {
		if err := s.Add(Conversation{SessionID: bad}); err == nil {
			t.Errorf("Add(%q) should fail", bad)
		}
		if _, err := Command(bad); err == nil {
			t.Errorf("Command(%q) should fail", bad)
		}
	}
	got, err := Command("0b9e7c1a-1234-4c5d-9e8f-abcdef012345")
	if err != nil || got != "claude --resume 0b9e7c1a-1234-4c5d-9e8f-abcdef012345" {
		t.Fatalf("Command = %q %v", got, err)
	}
}

func TestCorruptFileStartsEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "c.json")
	os.WriteFile(path, []byte("{nope"), 0o600)
	s, err := Open(path)
	if err == nil {
		t.Fatal("expected error")
	}
	if s == nil || len(s.List()) != 0 {
		t.Fatal("store should be usable and empty")
	}
}
