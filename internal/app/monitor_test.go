package app

import (
	"testing"

	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/procs"
	"github.com/brenobmoreira/skye/internal/tmuxctl"
	"github.com/brenobmoreira/skye/internal/winmem"
)

type fakeProcs struct{ list []procs.Proc }

func (f fakeProcs) Machine() (procs.Machine, error) {
	return procs.Machine{MemTotalMB: 10954, MemAvailMB: 8000}, nil
}
func (f fakeProcs) List() ([]procs.Proc, error) { return f.list, nil }

func TestMonitorMeasuresTerminalsAndFindsWhatWasLeftBehind(t *testing.T) {
	h := newHarness(t, config.Default())
	a, _ := h.app.NewTerminal("")
	b, _ := h.app.NewTerminal("")
	h.tmux.windows[0].PanePID = 100
	h.tmux.windows[1].PanePID = 200
	h.tmux.windows = append(h.tmux.windows, tmuxctl.Window{ID: "@9", Pane: "%9", SkyeID: "tstranger", PanePID: 900})
	mb := 1024
	h.app.o.PID = 50
	h.app.o.Procs = fakeProcs{list: []procs.Proc{
		{PID: 1, Name: "systemd"},
		{PID: 50, PPID: 1, Name: "skye", Args: "skye web --no-open", RSSKB: 40 * mb},
		{PID: 51, PPID: 50, Name: "tmux: client", RSSKB: 4 * mb},
		// terminal a: bash → claude → mcp
		{PID: 100, PPID: 60, Name: "bash", RSSKB: 10 * mb, Terminal: a.ID},
		{PID: 101, PPID: 100, Name: "claude", RSSKB: 300 * mb, Terminal: a.ID},
		{PID: 102, PPID: 101, Name: "mcp", RSSKB: 10 * mb, Terminal: a.ID},
		// terminal b: plain shell
		{PID: 200, PPID: 60, Name: "bash", RSSKB: 8 * mb, Terminal: b.ID},
		// the window nobody knows
		{PID: 900, PPID: 60, Name: "bash", RSSKB: 8 * mb, Terminal: "tstranger"},
		// a dev server that escaped a closed terminal, with a child
		{PID: 300, PPID: 1, Name: "node", Args: "node server.js", Cwd: "/home/demo/app", RSSKB: 200 * mb, Terminal: "tgone"},
		{PID: 301, PPID: 300, Name: "esbuild", RSSKB: 20 * mb, Terminal: "tgone"},
		// claude in some other terminal
		{PID: 400, PPID: 1, Name: "claude", Args: "claude", Cwd: "/home/demo/x", RSSKB: 350 * mb},
		// another skye
		{PID: 500, PPID: 1, Name: "skye", Args: "/home/demo/.local/bin/skye", RSSKB: 30 * mb},
	}}
	h.app.o.Servers = func() []tmuxctl.Server {
		return []tmuxctl.Server{{Name: "skye", Alive: true}, {Name: "skye-test-1", Alive: false}, {Name: "default", Alive: true}}
	}
	h.app.o.TmuxSocket = "skye"
	h.app.o.History = func() []procs.Point { return []procs.Point{{AvailMB: 7000}} }
	h.app.o.Host = func() (winmem.Memory, bool) { return winmem.Memory{TotalMB: 16044, AvailMB: 428}, true }

	r, err := h.app.Monitor()
	if err != nil {
		t.Fatal(err)
	}
	if r.Host == nil || r.Host.AvailMB != 428 {
		t.Fatalf("host = %+v", r.Host)
	}
	if r.Machine.MemAvailMB != 8000 || len(r.History) != 1 || r.SelfMB != 44 {
		t.Fatalf("machine=%+v history=%d self=%d", r.Machine, len(r.History), r.SelfMB)
	}
	loads := map[string]TerminalLoad{}
	for _, l := range r.Terminals {
		loads[l.ID] = l
	}
	if l := loads[a.ID]; l.RSSMB != 320 || l.Procs != 3 || l.PID != 100 {
		t.Fatalf("terminal a = %+v", l)
	}
	if l := loads[b.ID]; l.RSSMB != 8 || l.Procs != 1 {
		t.Fatalf("terminal b = %+v", l)
	}
	kinds := map[string]Stray{}
	for _, s := range r.Strays {
		kinds[s.Kind] = s
	}
	if len(r.Strays) != 4 {
		t.Fatalf("strays = %+v", r.Strays)
	}
	if s := kinds[StrayLoose]; s.PID != 300 || s.RSSMB != 220 || s.Procs != 2 || s.Terminal != "tgone" || s.Cwd != "/home/demo/app" {
		t.Fatalf("loose = %+v", s)
	}
	if s := kinds[StrayClaude]; s.PID != 400 || s.RSSMB != 350 {
		t.Fatalf("claude = %+v", s)
	}
	if s := kinds[StraySkye]; s.PID != 500 {
		t.Fatalf("skye = %+v", s)
	}
	if s := kinds[StrayWindow]; s.PID != 900 || s.Terminal != "tstranger" || s.RSSMB != 8 {
		t.Fatalf("window = %+v", s)
	}
	if len(r.Servers) != 2 || r.Servers[0].Name != "skye-test-1" || r.Servers[1].Name != "default" {
		t.Fatalf("servers = %+v", r.Servers)
	}
}

func TestMonitorWithoutAProcessSource(t *testing.T) {
	h := newHarness(t, config.Default())
	if _, err := h.app.Monitor(); err == nil {
		t.Fatal("monitor without /proc")
	}
	h.app.o.Procs = fakeProcs{}
	h.app.o.Host = func() (winmem.Memory, bool) { return winmem.Memory{}, false }
	if r, err := h.app.Monitor(); err != nil || r.Host != nil {
		t.Fatalf("host without a reading = %+v %v", r.Host, err)
	}
}
