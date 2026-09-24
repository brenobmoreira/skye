package terminals

import (
	"strings"
	"testing"
	"time"

	"github.com/brenobmoreira/skye/internal/hooks"
)

var t0 = time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC)

func newReg() *Registry {
	r := NewRegistry(func() time.Time { return t0 })
	r.Add(Terminal{ID: "a", Name: "blog", Window: "@1", Pane: "%1", State: Shell})
	return r
}

func apply(t *testing.T, r *Registry, name string, mods ...func(*hooks.Event)) Change {
	t.Helper()
	ev := hooks.Event{Terminal: "a", Name: name, SessionID: "s1"}
	for _, m := range mods {
		m(&ev)
	}
	ch, ok := r.Apply(ev)
	if !ok {
		t.Fatalf("%s not applied", name)
	}
	return ch
}

func TestHappyPathStatesAndAttention(t *testing.T) {
	r := newReg()
	steps := []struct {
		event     string
		state     State
		attention bool
	}{
		{"SessionStart", Idle, false},
		{"UserPromptSubmit", Running, false},
		{"Notification", Waiting, true},
		{"PostToolUse", Running, false},
		{"Stop", Idle, true},
	}
	for _, s := range steps {
		ch := apply(t, r, s.event, func(e *hooks.Event) { e.Prompt = "fix the header\nplease" })
		if ch.Terminal.State != s.state || ch.Attention != s.attention {
			t.Fatalf("%s: state=%s attention=%v", s.event, ch.Terminal.State, ch.Attention)
		}
	}
	got, _ := r.Get("a")
	if got.Title != "fix the header" || got.SessionID != "s1" {
		t.Fatalf("title=%q session=%q", got.Title, got.SessionID)
	}
}

func TestIdleReminderNotificationDoesNotBarkAgain(t *testing.T) {
	r := newReg()
	apply(t, r, "UserPromptSubmit")
	apply(t, r, "Stop")
	ch := apply(t, r, "Notification", func(e *hooks.Event) { e.Message = "Claude is waiting for your input" })
	if ch.Attention || ch.Terminal.State != Idle {
		t.Fatalf("reminder changed state: %+v", ch)
	}
	if ch := apply(t, r, "Stop"); ch.Attention {
		t.Fatal("second Stop while idle must not bark")
	}
}

func TestSessionEndArchivesAndStartsNewOne(t *testing.T) {
	r := newReg()
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "first task"; e.Cwd = "/home/demo/blog" })
	apply(t, r, "Stop")
	end := apply(t, r, "SessionEnd")
	if end.Ended == nil || end.Ended.SessionID != "s1" || end.Ended.Title != "first task" || end.Ended.Cwd != "/home/demo/blog" || !end.Ended.EndedAt.Equal(t0) {
		t.Fatalf("ended = %+v", end.Ended)
	}
	if end.Terminal.State != Shell || end.Terminal.SessionID != "" || end.Terminal.Title != "" {
		t.Fatalf("after end: %+v", end.Terminal)
	}
	start := apply(t, r, "SessionStart", func(e *hooks.Event) { e.SessionID = "s2" })
	if start.Ended != nil || start.Terminal.SessionID != "s2" {
		t.Fatalf("start = %+v", start)
	}
}

func TestSessionStartWithNewIDEndsPrevious(t *testing.T) {
	r := newReg()
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "old" })
	ch := apply(t, r, "SessionStart", func(e *hooks.Event) { e.SessionID = "s2" })
	if ch.Ended == nil || ch.Ended.SessionID != "s1" || ch.Terminal.Title != "" {
		t.Fatalf("change = %+v", ch)
	}
	same := apply(t, r, "SessionStart", func(e *hooks.Event) { e.SessionID = "s2" })
	if same.Ended != nil {
		t.Fatal("same session id must not end the conversation")
	}
}

func TestSessionStartPreservesOldCwdInEndedConversation(t *testing.T) {
	r := newReg()
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "first"; e.Cwd = "/home/demo/old" })
	ch := apply(t, r, "SessionStart", func(e *hooks.Event) { e.SessionID = "s2"; e.Cwd = "/home/demo/new" })
	if ch.Ended == nil || ch.Ended.Cwd != "/home/demo/old" {
		t.Fatalf("ended.cwd = %q, want /home/demo/old", ch.Ended.Cwd)
	}
	if ch.Terminal.Cwd != "/home/demo/new" {
		t.Fatalf("terminal.cwd = %q, want /home/demo/new", ch.Terminal.Cwd)
	}
}

func TestUnknownTerminalOrEventIsIgnored(t *testing.T) {
	r := newReg()
	if _, ok := r.Apply(hooks.Event{Terminal: "zzz", Name: "Stop"}); ok {
		t.Fatal("unknown terminal applied")
	}
	if _, ok := r.Apply(hooks.Event{Terminal: "a", Name: "PreCompact", Cwd: "/elsewhere"}); ok {
		t.Fatal("unknown event applied")
	}
	term, _ := r.Get("a")
	if term.Cwd != "" {
		t.Fatalf("unknown event mutated cwd to %q", term.Cwd)
	}
}

func TestTitleFirstLineAndLimit(t *testing.T) {
	r := newReg()
	long := strings.Repeat("é", TitleLimit+10)
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "\n\n  " + long })
	got, _ := r.Get("a")
	if n := len([]rune(got.Title)); n != TitleLimit {
		t.Fatalf("title runes = %d", n)
	}
	if !strings.HasSuffix(got.Title, "…") {
		t.Fatalf("title = %q", got.Title)
	}
}

func TestListOrderLookupRenameRemove(t *testing.T) {
	now := t0
	r := NewRegistry(func() time.Time { now = now.Add(time.Second); return now })
	r.Add(Terminal{ID: "shell1", Pane: "%1", Window: "@1", State: Shell})
	r.Add(Terminal{ID: "run1", Pane: "%2", Window: "@2", State: Running})
	r.Add(Terminal{ID: "wait1", Pane: "%3", Window: "@3", State: Waiting})
	r.Add(Terminal{ID: "idle1", Pane: "%4", Window: "@4", State: Idle})
	r.Add(Terminal{ID: "wait2", Pane: "%5", Window: "@5", State: Waiting})
	var ids []string
	for _, t := range r.List() {
		ids = append(ids, t.ID)
	}
	if strings.Join(ids, ",") != "wait1,wait2,idle1,run1,shell1" {
		t.Fatalf("order = %v", ids)
	}
	if tt, ok := r.ByPane("%4"); !ok || tt.ID != "idle1" {
		t.Fatal("ByPane")
	}
	if tt, ok := r.ByWindow("@2"); !ok || tt.ID != "run1" {
		t.Fatal("ByWindow")
	}
	if _, ok := r.Rename("run1", "   "); ok {
		t.Fatal("blank rename accepted")
	}
	if tt, ok := r.Rename("run1", "  api  "); !ok || tt.Name != "api" {
		t.Fatalf("rename = %+v", tt)
	}
	if _, ok := r.Remove("run1"); !ok {
		t.Fatal("remove")
	}
	if _, ok := r.Get("run1"); ok {
		t.Fatal("still there")
	}
	if len(NewRegistry(time.Now).List()) != 0 || NewRegistry(time.Now).List() == nil {
		t.Fatal("empty list must be non-nil")
	}
}

func ids(list []Terminal) string {
	var out []string
	for _, t := range list {
		out = append(out, t.ID)
	}
	return strings.Join(out, ",")
}

func TestNewTerminalsGoToTheEndOfTheirGroup(t *testing.T) {
	r := NewRegistry(func() time.Time { return t0 })
	r.Add(Terminal{ID: "b", State: Idle})
	r.Add(Terminal{ID: "a", State: Idle})
	r.Add(Terminal{ID: "c", State: Idle, Order: -5})
	if got := ids(r.List()); got != "c,b,a" {
		t.Fatalf("order = %s", got)
	}
}

func TestTerminalThatStopsRunningGoesToTheTopOfItsGroup(t *testing.T) {
	r := NewRegistry(func() time.Time { return t0 })
	for _, id := range []string{"i1", "i2", "a", "w1"} {
		r.Add(Terminal{ID: id, State: Idle})
	}
	r.Apply(hooks.Event{Terminal: "w1", Name: "UserPromptSubmit"})
	r.Apply(hooks.Event{Terminal: "w1", Name: "Notification"})
	r.Apply(hooks.Event{Terminal: "a", Name: "UserPromptSubmit"})
	if got := ids(r.List()); got != "w1,i1,i2,a" {
		t.Fatalf("while running = %s", got)
	}
	r.Apply(hooks.Event{Terminal: "a", Name: "Stop"})
	if got := ids(r.List()); got != "w1,a,i1,i2" {
		t.Fatalf("after stop = %s", got)
	}
}

func TestIdleTerminalDoesNotJumpOnRepeatedStop(t *testing.T) {
	r := NewRegistry(func() time.Time { return t0 })
	r.Add(Terminal{ID: "x", State: Idle})
	r.Add(Terminal{ID: "y", State: Idle})
	r.Apply(hooks.Event{Terminal: "y", Name: "Stop"})
	if got := ids(r.List()); got != "x,y" {
		t.Fatalf("order = %s", got)
	}
}

func TestReorderFollowsGivenIDsAndKeepsTheRest(t *testing.T) {
	r := NewRegistry(func() time.Time { return t0 })
	for _, id := range []string{"a", "b", "c", "d"} {
		r.Add(Terminal{ID: id, State: Idle})
	}
	changed := r.Reorder([]string{"c", "a", "ghost"})
	if got := ids(r.List()); got != "c,a,b,d" {
		t.Fatalf("order = %s", got)
	}
	if len(changed) != 4 {
		t.Fatalf("changed = %v", changed)
	}
	r.Add(Terminal{ID: "e", State: Idle})
	if got := ids(r.List()); got != "c,a,b,d,e" {
		t.Fatalf("after add = %s", got)
	}
}

func TestReorderCannotMoveAcrossStateGroups(t *testing.T) {
	r := NewRegistry(func() time.Time { return t0 })
	r.Add(Terminal{ID: "w", State: Waiting})
	r.Add(Terminal{ID: "i", State: Idle})
	r.Reorder([]string{"i", "w"})
	if got := ids(r.List()); got != "w,i" {
		t.Fatalf("order = %s", got)
	}
}

func TestClearKeepsTheSessionOutOfEnded(t *testing.T) {
	r := newReg()
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "first task" })
	apply(t, r, "Stop")
	end := apply(t, r, "SessionEnd", func(e *hooks.Event) { e.Reason = "clear" })
	if end.Ended != nil || end.Terminal.State != Idle {
		t.Fatalf("clear ended the session: %+v", end)
	}
	start := apply(t, r, "SessionStart", func(e *hooks.Event) { e.SessionID = "s2"; e.Source = "clear" })
	if start.Ended != nil || start.Terminal.SessionID != "s2" || start.Terminal.Title != "" || start.Terminal.State != Idle {
		t.Fatalf("start after clear = %+v", start)
	}
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.SessionID = "s2"; e.Prompt = "second task" })
	if got, _ := r.Get("a"); got.Title != "second task" {
		t.Fatalf("title = %q", got.Title)
	}
}

func TestClearStartWithoutEndDoesNotArchive(t *testing.T) {
	r := newReg()
	apply(t, r, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "old" })
	ch := apply(t, r, "SessionStart", func(e *hooks.Event) { e.SessionID = "s2"; e.Source = "clear" })
	if ch.Ended != nil || ch.Terminal.Title != "" {
		t.Fatalf("change = %+v", ch)
	}
}
