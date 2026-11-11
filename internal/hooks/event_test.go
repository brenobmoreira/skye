package hooks

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestParseReadsCompactionAndSubagents(t *testing.T) {
	pre, _ := Parse("t1", []byte(`{"hook_event_name":"PreCompact","trigger":"auto"}`))
	sub, _ := Parse("t1", []byte(`{"hook_event_name":"SubagentStart","agent_id":"agent-1","agent_type":"Explore"}`))
	if pre.Trigger != "auto" || sub.AgentID != "agent-1" {
		t.Fatalf("pre = %+v, sub = %+v", pre, sub)
	}
}

func TestParseEventSupportsAgyPayloads(t *testing.T) {
	body := []byte(`{"conversationId":"conv-9","workspacePaths":["/home/demo/skye"],"lastUserInput":"adicione agy","modelName":"Gemini 2.5 Pro"}`)
	ev, err := ParseEvent("t1", "PreInvocation", body)
	if err != nil {
		t.Fatal(err)
	}
	if ev.Name != "UserPromptSubmit" || ev.Provider != "agy" || ev.SessionID != "conv-9" || ev.Cwd != "/home/demo/skye" || ev.Prompt != "adicione agy" || ev.Context == nil || ev.Context.Model != "Gemini 2.5 Pro" {
		t.Fatalf("got %+v", ev)
	}
}

func TestParseReadsClaudeLastMessage(t *testing.T) {
	ev, err := Parse("t1", []byte(`{"hook_event_name":"Stop","last_assistant_message":"Faço o push?"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.LastMessage != "Faço o push?" {
		t.Fatalf("got %+v", ev)
	}
}

func writeTranscript(t *testing.T, lines ...string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "rollout.jsonl")
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const codexTokenCount = `{"type":"event_msg","payload":{"type":"token_count","info":{"last_token_usage":{"total_tokens":1000},"model_context_window":10000}}}`

func TestParseReadsCodexLastMessage(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"event_msg","payload":{"type":"task_complete","last_agent_message":"Primeiro turno."}}`,
		codexTokenCount,
		`{"type":"event_msg","payload":{"type":"task_complete","last_agent_message":"Posso commitar?"}}`,
	)
	ev, err := Parse("t1", []byte(`{"hook_event_name":"Stop","model":"gpt-5","transcript_path":"`+path+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.LastMessage != "Posso commitar?" || ev.Provider != "codex" {
		t.Fatalf("got %+v", ev)
	}
}

func TestParseCodexWithoutTaskComplete(t *testing.T) {
	path := writeTranscript(t, codexTokenCount)
	ev, err := Parse("t1", []byte(`{"hook_event_name":"Stop","model":"gpt-5","transcript_path":"`+path+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.LastMessage != "" || ev.Context == nil || ev.Context.ContextPct == nil || *ev.Context.ContextPct != 10 {
		t.Fatalf("got %+v", ev)
	}
}

func TestParseCodexIgnoresTheReplyOfAnEarlierTurn(t *testing.T) {
	path := writeTranscript(t,
		`{"type":"event_msg","payload":{"type":"task_complete","last_agent_message":"Posso commitar?"}}`,
		`{"type":"event_msg","payload":{"type":"task_started"}}`,
		`{"type":"event_msg","payload":{"type":"user_message","message":"rode os testes"}}`,
		codexTokenCount,
	)
	ev, err := Parse("t1", []byte(`{"hook_event_name":"Stop","model":"gpt-5","transcript_path":"`+path+`"}`))
	if err != nil {
		t.Fatal(err)
	}
	if ev.LastMessage != "" {
		t.Fatalf("stale reply %q", ev.LastMessage)
	}
}
