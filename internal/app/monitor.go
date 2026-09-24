package app

import (
	"errors"
	"sort"

	"github.com/brenobmoreira/skye/internal/procs"
	"github.com/brenobmoreira/skye/internal/tmuxctl"
)

// Kinds of things the monitor finds outside the list of terminals.
const (
	StrayLoose  = "solto"  // inherited SKYE_TERMINAL_ID but lives under no pane
	StrayClaude = "claude" // claude started outside skye
	StraySkye   = "skye"   // another skye process
	StrayWindow = "janela" // tmux window the list does not know
)

type MonitorReport struct {
	Machine   procs.Machine    `json:"machine"`
	History   []procs.Point    `json:"history"`
	SelfMB    int              `json:"selfMb"`
	Terminals []TerminalLoad   `json:"terminals"`
	Strays    []Stray          `json:"strays"`
	Servers   []tmuxctl.Server `json:"servers"`
}

type TerminalLoad struct {
	ID    string `json:"id"`
	PID   int    `json:"pid"`
	Procs int    `json:"procs"`
	RSSMB int    `json:"rssMb"`
}

type Stray struct {
	Kind     string `json:"kind"`
	PID      int    `json:"pid"`
	Name     string `json:"name"`
	Args     string `json:"args"`
	Cwd      string `json:"cwd"`
	Terminal string `json:"terminal,omitempty"`
	Procs    int    `json:"procs"`
	RSSMB    int    `json:"rssMb"`
}

// Monitor measures the machine and each terminal's process tree, and lists what looks left
// behind: processes and windows no terminal in the list accounts for.
func (a *App) Monitor() (MonitorReport, error) {
	if a.o.Procs == nil {
		return MonitorReport{}, errors.New("monitor indisponível")
	}
	var r MonitorReport
	var err error
	if r.Machine, err = a.o.Procs.Machine(); err != nil {
		return r, err
	}
	list, err := a.o.Procs.List()
	if err != nil {
		return r, err
	}
	windows, err := a.o.Tmux.ListWindows()
	if err != nil {
		return r, err
	}
	table := procs.Index(list)
	accounted := map[int]bool{}
	claim := func(pids []int) {
		for _, pid := range pids {
			accounted[pid] = true
		}
	}
	self := table.Tree(a.o.PID)
	claim(self)
	r.SelfMB = table.RSSKB(self) / 1024

	known := map[string]bool{}
	for _, t := range a.reg.List() {
		known[t.ID] = true
	}
	for _, w := range windows {
		tree := table.Tree(w.PanePID)
		claim(tree)
		if known[w.SkyeID] {
			r.Terminals = append(r.Terminals, TerminalLoad{ID: w.SkyeID, PID: w.PanePID, Procs: len(tree), RSSMB: table.RSSKB(tree) / 1024})
			continue
		}
		r.Strays = append(r.Strays, stray(table, StrayWindow, w.PanePID, tree, w.SkyeID))
	}

	loose := map[int]bool{}
	for _, p := range list {
		if p.Terminal != "" && !accounted[p.PID] {
			loose[p.PID] = true
		}
	}
	pids := make([]int, 0, len(list))
	for _, p := range list {
		pids = append(pids, p.PID)
	}
	sort.Ints(pids)
	for _, pid := range pids {
		p := table[pid]
		switch {
		case accounted[pid]:
		case loose[pid] && !loose[p.PPID]:
			tree := table.Tree(pid)
			claim(tree)
			r.Strays = append(r.Strays, stray(table, StrayLoose, pid, tree, p.Terminal))
		case p.Terminal == "" && p.Name == "claude":
			tree := table.Tree(pid)
			claim(tree)
			r.Strays = append(r.Strays, stray(table, StrayClaude, pid, tree, ""))
		case p.Terminal == "" && p.Name == "skye":
			tree := table.Tree(pid)
			claim(tree)
			r.Strays = append(r.Strays, stray(table, StraySkye, pid, tree, ""))
		}
	}

	if a.o.Servers != nil {
		for _, s := range a.o.Servers() {
			if s.Name != a.o.TmuxSocket {
				r.Servers = append(r.Servers, s)
			}
		}
	}
	if a.o.History != nil {
		r.History = a.o.History()
	}
	return r, nil
}

func stray(table procs.Table, kind string, pid int, tree []int, terminal string) Stray {
	p := table[pid]
	return Stray{Kind: kind, PID: pid, Name: p.Name, Args: p.Args, Cwd: p.Cwd, Terminal: terminal, Procs: len(tree), RSSMB: table.RSSKB(tree) / 1024}
}
