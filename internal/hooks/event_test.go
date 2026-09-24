package hooks

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	body := []byte(`{"hook_event_name":"UserPromptSubmit","session_id":"abc-1","cwd":"/home/demo/blog","prompt":"fix the header","transcript_path":"/x"}`)
	ev, err := Parse("t1", body)
	if err != nil {
		t.Fatal(err)
	}
	want := Event{Terminal: "t1", Name: "UserPromptSubmit", SessionID: "abc-1", Cwd: "/home/demo/blog", Prompt: "fix the header"}
	if ev != want {
		t.Fatalf("got %+v", ev)
	}
}

func TestParseReadsWhyASessionStartsOrEnds(t *testing.T) {
	end, _ := Parse("t1", []byte(`{"hook_event_name":"SessionEnd","reason":"clear"}`))
	start, _ := Parse("t1", []byte(`{"hook_event_name":"SessionStart","source":"clear"}`))
	if end.Reason != "clear" || start.Source != "clear" {
		t.Fatalf("end = %+v, start = %+v", end, start)
	}
}

func TestParseRejects(t *testing.T) {
	if _, err := Parse("", []byte(`{"hook_event_name":"Stop"}`)); !errors.Is(err, ErrNoTerminal) {
		t.Fatalf("no terminal: %v", err)
	}
	for _, body := range []string{`not json`, `{}`} {
		if _, err := Parse("t1", []byte(body)); err == nil {
			t.Errorf("Parse(%q) should fail", body)
		}
	}
}
