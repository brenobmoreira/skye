package app

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/launch"
	"github.com/brenobmoreira/skye/internal/resume"
	"github.com/brenobmoreira/skye/internal/terminals"
	"github.com/brenobmoreira/skye/internal/tmuxctl"
)

type fakeTmux struct {
	mu       sync.Mutex
	next     int
	windows  []tmuxctl.Window
	created  [][]string
	envs     []map[string]string
	keys     map[string]string
	killed   []string
	serverKO bool
}

func (f *fakeTmux) NewWindow(id string, env map[string]string, argv []string) (tmuxctl.Window, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.next++
	w := tmuxctl.Window{ID: fmt.Sprintf("@%d", f.next), Pane: fmt.Sprintf("%%%d", f.next), SkyeID: id}
	f.windows = append(f.windows, w)
	f.created = append(f.created, argv)
	f.envs = append(f.envs, env)
	return w, nil
}
func (f *fakeTmux) ListWindows() ([]tmuxctl.Window, error) { return f.windows, nil }
func (f *fakeTmux) SendKeys(pane string, data []byte) error {
	f.keys[pane] += string(data)
	return nil
}
func (f *fakeTmux) Capture(pane string) (string, error) { return "screen of " + pane, nil }
func (f *fakeTmux) Resize(string, int, int) error       { return nil }
func (f *fakeTmux) KillWindow(w string) error           { f.killed = append(f.killed, w); return nil }
func (f *fakeTmux) KillServer() error                   { f.serverKO = true; return nil }

type fakeNotifier struct {
	mu    sync.Mutex
	shown []string
}

func (n *fakeNotifier) Show(title, body string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.shown = append(n.shown, title+"|"+body)
	return nil
}
func (n *fakeNotifier) count() int { n.mu.Lock(); defer n.mu.Unlock(); return len(n.shown) }

type harness struct {
	app    *App
	tmux   *fakeTmux
	notif  *fakeNotifier
	store  *resume.Store
	paths  config.Paths
	mu     sync.Mutex
	events []string
	barks  []string
}

func newHarness(t *testing.T, cfg config.Config) *harness {
	t.Helper()
	dir := t.TempDir()
	paths := config.Paths{
		LaunchDir:     filepath.Join(dir, "launch"),
		Conversations: filepath.Join(dir, "conversations.json"),
		Socket:        filepath.Join(dir, "skye.sock"),
	}
	store, _ := resume.Open(paths.Conversations)
	h := &harness{tmux: &fakeTmux{keys: map[string]string{}}, notif: &fakeNotifier{}, store: store, paths: paths}
	ids := 0
	h.app = New(Options{
		Config:   cfg,
		Paths:    paths,
		Tmux:     h.tmux,
		Store:    store,
		Notifier: h.notif,
		Emit: func(ev string, data any) {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.events = append(h.events, ev)
			if ev == "bark" {
				h.barks = append(h.barks, data.(string))
			}
		},
		Logf:     t.Logf,
		SelfPath: "/usr/local/bin/skye",
		Home:     "/home/demo",
		NewID:    func() string { ids++; return fmt.Sprintf("t%d", ids) },
		Now:      func() time.Time { return time.Date(2026, 9, 23, 12, 0, 0, 0, time.UTC) },
	})
	return h
}

func (h *harness) hook(id, name string, mods ...func(*hooks.Event)) {
	ev := hooks.Event{Terminal: id, Name: name, SessionID: "0b9e7c1a-1234"}
	for _, m := range mods {
		m(&ev)
	}
	h.app.HandleHook(ev)
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("condition not met")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestNewTerminalPlainAndPreset(t *testing.T) {
	h := newHarness(t, config.Config{Sound: true, Presets: []config.Preset{{Name: "blog", Command: "cd ~/projects/blog && claude"}}})
	plain, err := h.app.NewTerminal("")
	if err != nil {
		t.Fatal(err)
	}
	if plain.Name != "shell" || plain.State != terminals.Shell {
		t.Fatalf("plain = %+v", plain)
	}
	if got := strings.Join(h.tmux.created[0], " "); got != "/usr/local/bin/skye launch t1" {
		t.Fatalf("argv = %s", got)
	}
	if h.tmux.envs[0]["SKYE_TERMINAL_ID"] != "t1" || h.tmux.envs[0]["SKYE_SOCKET"] != h.paths.Socket {
		t.Fatalf("env = %v", h.tmux.envs[0])
	}
	blog, err := h.app.NewTerminal("blog")
	if err != nil {
		t.Fatal(err)
	}
	spec, err := launch.Read(h.paths.LaunchDir, blog.ID)
	if err != nil || spec.Command != "cd ~/projects/blog && claude" || spec.Name != "blog" || spec.Cwd != "/home/demo" {
		t.Fatalf("spec = %+v %v", spec, err)
	}
	if _, err := h.app.NewTerminal("nope"); err == nil {
		t.Fatal("unknown preset accepted")
	}
	if len(h.app.List()) != 2 {
		t.Fatalf("list = %+v", h.app.List())
	}
}

func TestInputRouting(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	if err := h.app.Write(term.ID, "ls\r"); err != nil {
		t.Fatal(err)
	}
	if h.tmux.keys["%1"] != "ls\r" {
		t.Fatalf("keys=%q", h.tmux.keys)
	}
	if snap, _ := h.app.Snapshot(term.ID); snap != "screen of %1" {
		t.Fatalf("snapshot = %q", snap)
	}
	if err := h.app.Write("missing", "x"); err == nil {
		t.Fatal("unknown terminal accepted")
	}
}

func TestAttentionBarksAndToastsOnlyWhenUnfocused(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.app.SetFocused(true)
	h.hook(term.ID, "UserPromptSubmit")
	h.hook(term.ID, "Stop")
	if len(h.barks) != 1 || h.barks[0] != term.ID {
		t.Fatalf("barks = %v", h.barks)
	}
	time.Sleep(50 * time.Millisecond)
	if h.notif.count() != 0 {
		t.Fatal("toast shown while focused")
	}
	h.app.SetFocused(false)
	h.hook(term.ID, "UserPromptSubmit")
	h.hook(term.ID, "Notification")
	waitFor(t, func() bool { return h.notif.count() == 1 })
	if h.notif.shown[0] != "shell|esperando você" {
		t.Fatalf("toast = %v", h.notif.shown)
	}
	h.app.SetSound(false)
	h.hook(term.ID, "Stop")
	if len(h.barks) != 2 {
		t.Fatalf("bark with sound off: %v", h.barks)
	}
}

func TestCloseArchivesConversation(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.hook(term.ID, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "write tests"; e.Cwd = "/home/demo/blog" })
	if err := h.app.Close(term.ID); err != nil {
		t.Fatal(err)
	}
	if len(h.tmux.killed) != 1 || len(h.app.List()) != 0 {
		t.Fatalf("killed=%v list=%v", h.tmux.killed, h.app.List())
	}
	convs := h.app.Conversations()
	if len(convs) != 1 || convs[0].Title != "write tests" || convs[0].Cwd != "/home/demo/blog" {
		t.Fatalf("conversations = %+v", convs)
	}
	if _, err := launch.Read(h.paths.LaunchDir, term.ID); err == nil {
		t.Fatal("launch spec not removed")
	}
}

func TestClearKeepsConversationOutOfEnded(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.hook(term.ID, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "old task" })
	h.hook(term.ID, "SessionEnd", func(e *hooks.Event) { e.Reason = "clear" })
	h.hook(term.ID, "SessionStart", func(e *hooks.Event) { e.SessionID = "new-id"; e.Source = "clear" })
	if c := h.app.Conversations(); len(c) != 0 {
		t.Fatalf("conversations = %+v", c)
	}
	if got := h.app.List()[0]; got.SessionID != "new-id" || got.Title != "" {
		t.Fatalf("terminal = %+v", got)
	}
}

func TestExitMovesConversationToEnded(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.hook(term.ID, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "old task" })
	h.hook(term.ID, "SessionEnd", func(e *hooks.Event) { e.Reason = "prompt_input_exit" })
	if c := h.app.Conversations(); len(c) != 1 || c[0].Title != "old task" {
		t.Fatalf("conversations = %+v", c)
	}
}

func TestWindowClosedByShellExit(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.hook(term.ID, "UserPromptSubmit", func(e *hooks.Event) { e.Prompt = "bye" })
	h.app.HandleWindowClosed("@1")
	if len(h.app.List()) != 0 || len(h.app.Conversations()) != 1 {
		t.Fatalf("list=%v convs=%v", h.app.List(), h.app.Conversations())
	}
}

func TestResume(t *testing.T) {
	h := newHarness(t, config.Default())
	h.store.Add(resume.Conversation{SessionID: "abc-123", Cwd: "/home/demo/blog", Title: "old task"})
	term, err := h.app.Resume("abc-123")
	if err != nil {
		t.Fatal(err)
	}
	spec, _ := launch.Read(h.paths.LaunchDir, term.ID)
	if spec.Command != "claude --resume abc-123" || spec.Cwd != "/home/demo/blog" || term.Name != "old task" {
		t.Fatalf("spec=%+v term=%+v", spec, term)
	}
	if len(h.app.Conversations()) != 0 {
		t.Fatal("resumed conversation still listed")
	}
	if _, err := h.app.Resume("missing"); err == nil {
		t.Fatal("unknown conversation resumed")
	}
}

func TestRecoverRestoresTerminalsAndCleansOrphans(t *testing.T) {
	h := newHarness(t, config.Default())
	launch.Write(h.paths.LaunchDir, "kept", launch.Spec{Name: "api", Cwd: "/home/demo/api"})
	launch.Write(h.paths.LaunchDir, "orphan", launch.Spec{Name: "gone"})
	h.tmux.windows = []tmuxctl.Window{
		{ID: "@3", Pane: "%3", SkyeID: "kept"},
		{ID: "@4", Pane: "%4", SkyeID: "nospec"},
	}
	if err := h.app.Recover(); err != nil {
		t.Fatal(err)
	}
	list := h.app.List()
	if len(list) != 2 {
		t.Fatalf("list = %+v", list)
	}
	names := map[string]string{}
	for _, tt := range list {
		names[tt.ID] = tt.Name
	}
	if names["kept"] != "api" || names["nospec"] != "shell" {
		t.Fatalf("names = %v", names)
	}
	if _, err := launch.Read(h.paths.LaunchDir, "orphan"); err == nil {
		t.Fatal("orphan launch spec kept")
	}
}

func TestOutputRoutingAndRename(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.app.HandleOutput("%1", []byte("hi"))
	h.app.HandleOutput("%99", []byte("ignored"))
	count := 0
	for _, e := range h.events {
		if e == "output" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("output events = %d", count)
	}
	if err := h.app.Rename(term.ID, "notes"); err != nil {
		t.Fatal(err)
	}
	spec, _ := launch.Read(h.paths.LaunchDir, term.ID)
	if spec.Name != "notes" || h.app.List()[0].Name != "notes" {
		t.Fatalf("rename not persisted: %+v", spec)
	}
	if err := h.app.Rename(term.ID, " "); err == nil {
		t.Fatal("blank rename accepted")
	}
}

func TestQuitArchivesAllAndKillsServer(t *testing.T) {
	h := newHarness(t, config.Default())
	a, _ := h.app.NewTerminal("")
	b, _ := h.app.NewTerminal("")
	h.hook(a.ID, "UserPromptSubmit", func(e *hooks.Event) { e.SessionID = "sa"; e.Prompt = "a" })
	h.hook(b.ID, "UserPromptSubmit", func(e *hooks.Event) { e.SessionID = "sb"; e.Prompt = "b" })
	if err := h.app.Quit(); err != nil {
		t.Fatal(err)
	}
	if !h.tmux.serverKO || len(h.app.Conversations()) != 2 {
		t.Fatalf("server killed=%v convs=%v", h.tmux.serverKO, h.app.Conversations())
	}
	ids, _ := launch.List(h.paths.LaunchDir)
	if len(ids) != 0 {
		t.Fatalf("launch specs left: %v", ids)
	}
}

func TestReorderPersistsAndSurvivesRecover(t *testing.T) {
	h := newHarness(t, config.Default())
	a, _ := h.app.NewTerminal("")
	b, _ := h.app.NewTerminal("")
	if err := h.app.Reorder([]string{b.ID, a.ID}); err != nil {
		t.Fatal(err)
	}
	if got := h.app.List(); got[0].ID != b.ID {
		t.Fatalf("list = %+v", got)
	}
	spec, _ := launch.Read(h.paths.LaunchDir, b.ID)
	if spec.Order != 1 {
		t.Fatalf("order not persisted: %+v", spec)
	}

	again := newHarness(t, config.Default())
	again.paths = h.paths
	again.app.o.Paths = h.paths
	again.tmux.windows = []tmuxctl.Window{{ID: "@1", Pane: "%1", SkyeID: a.ID}, {ID: "@2", Pane: "%2", SkyeID: b.ID}}
	if err := again.app.Recover(); err != nil {
		t.Fatal(err)
	}
	if got := again.app.List(); got[0].ID != b.ID {
		t.Fatalf("recovered list = %+v", got)
	}
}

func TestUsageKeepsTheLatestAndEmitsOnlyOnChange(t *testing.T) {
	h := newHarness(t, config.Default())
	if u := h.app.Usage(); u.FiveHour != nil {
		t.Fatalf("usage before any status line = %+v", u)
	}
	u := hooks.Usage{FiveHour: &hooks.Window{UsedPct: 40, ResetsAt: 1790000000}}
	h.app.HandleUsage(u)
	h.app.HandleUsage(hooks.Usage{FiveHour: &hooks.Window{UsedPct: 40, ResetsAt: 1790000000}})
	h.app.HandleUsage(hooks.Usage{FiveHour: &hooks.Window{UsedPct: 41, ResetsAt: 1790000000}})
	count := 0
	for _, e := range h.events {
		if e == "usage" {
			count++
		}
	}
	if count != 2 {
		t.Fatalf("usage events = %d", count)
	}
	if got := h.app.Usage(); got.FiveHour.UsedPct != 41 || got.UpdatedAt.IsZero() {
		t.Fatalf("usage = %+v", got)
	}
}

func TestToastSaysWhatTheTerminalAsks(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	h.app.SetFocused(false)
	h.hook(term.ID, "UserPromptSubmit")
	h.hook(term.ID, "Notification", func(e *hooks.Event) { e.Message = "Claude needs your permission to use Bash" })
	waitFor(t, func() bool { return h.notif.count() == 1 })
	if h.notif.shown[0] != "shell|Claude needs your permission to use Bash" {
		t.Fatalf("toast = %v", h.notif.shown)
	}
}

func TestStatusLineFeedsUsageAndTheTerminalContext(t *testing.T) {
	h := newHarness(t, config.Default())
	term, _ := h.app.NewTerminal("")
	pct := 30.0
	s := hooks.Status{HasUsage: true, Usage: hooks.Usage{FiveHour: &hooks.Window{UsedPct: 5}}, Session: hooks.Session{ContextPct: &pct, Model: "Opus 5.5"}}
	before := len(h.events)
	h.app.HandleStatus(term.ID, s)
	h.app.HandleStatus(term.ID, s)
	counts := map[string]int{}
	for _, e := range h.events[before:] {
		counts[e]++
	}
	if counts["terminals"] != 1 || counts["usage"] != 1 {
		t.Fatalf("events = %v", counts)
	}
	if got := h.app.List()[0].Context; got == nil || *got.ContextPct != 30 {
		t.Fatalf("context = %+v", got)
	}
}
