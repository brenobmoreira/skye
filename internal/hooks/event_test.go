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

func TestParseDescribesThePermissionRequest(t *testing.T) {
	cases := map[string]string{
		`{"tool_name":"Bash","tool_input":{"command":"git push origin main","description":"push"}}`: "Bash: git push origin main",
		`{"tool_name":"Edit","tool_input":{"file_path":"/home/demo/blog/a.go"}}`:                    "Edit: /home/demo/blog/a.go",
		`{"tool_name":"WebFetch","tool_input":{"url":"https://example.com"}}`:                       "WebFetch: https://example.com",
		`{"tool_name":"mcp__x__y","tool_input":{"q":1}}`:                                            "mcp__x__y",
		`{"x":1}`: "",
	}
	for extra, want := range cases {
		body := `{"hook_event_name":"PermissionRequest",` + extra[1:]
		ev, err := Parse("t1", []byte(body))
		if err != nil {
			t.Fatal(err)
		}
		if ev.Tool != want {
			t.Errorf("%s: tool = %q, want %q", extra, ev.Tool, want)
		}
	}
}
