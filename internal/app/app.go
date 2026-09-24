package app

import (
	"encoding/base64"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/launch"
	"github.com/brenobmoreira/skye/internal/resume"
	"github.com/brenobmoreira/skye/internal/terminals"
	"github.com/brenobmoreira/skye/internal/tmuxctl"
)

type Tmux interface {
	NewWindow(skyeID string, env map[string]string, argv []string) (tmuxctl.Window, error)
	ListWindows() ([]tmuxctl.Window, error)
	SendKeys(pane string, data []byte) error
	Capture(pane string) (string, error)
	Resize(window string, cols, rows int) error
	KillWindow(window string) error
	KillServer() error
}

type Notifier interface {
	Show(title, body string) error
}

type Options struct {
	Config   config.Config
	Paths    config.Paths
	Tmux     Tmux
	Store    *resume.Store
	Notifier Notifier
	Emit     func(event string, data any)
	Logf     func(format string, args ...any)
	SelfPath string
	Home     string
	NewID    func() string
	Now      func() time.Time
}

type OutputEvent struct {
	ID   string `json:"id"`
	Data string `json:"data"`
}

type App struct {
	o       Options
	reg     *terminals.Registry
	mu      sync.Mutex
	sound   bool
	focused bool
	usage   hooks.Usage
}

func New(o Options) *App {
	return &App{o: o, reg: terminals.NewRegistry(o.Now), sound: o.Config.Sound}
}

func (a *App) Recover() error {
	windows, err := a.o.Tmux.ListWindows()
	if err != nil {
		return err
	}
	alive := map[string]bool{}
	for _, w := range windows {
		alive[w.SkyeID] = true
		spec, err := launch.Read(a.o.Paths.LaunchDir, w.SkyeID)
		if err != nil {
			a.o.Logf("recover %s: %v", w.SkyeID, err)
			spec = launch.Spec{Name: "shell"}
		}
		a.reg.Add(terminals.Terminal{ID: w.SkyeID, Name: spec.Name, Preset: spec.Preset, Cwd: spec.Cwd, Window: w.ID, Pane: w.Pane, Order: spec.Order})
	}
	ids, err := launch.List(a.o.Paths.LaunchDir)
	if err != nil {
		a.o.Logf("list launch specs: %v", err)
	}
	for _, id := range ids {
		if !alive[id] {
			_ = launch.Remove(a.o.Paths.LaunchDir, id)
		}
	}
	a.emitTerminals()
	return nil
}

func (a *App) open(spec launch.Spec) (terminals.Terminal, error) {
	id := a.o.NewID()
	if err := launch.Write(a.o.Paths.LaunchDir, id, spec); err != nil {
		return terminals.Terminal{}, err
	}
	env := map[string]string{"SKYE_TERMINAL_ID": id, "SKYE_SOCKET": a.o.Paths.Socket}
	w, err := a.o.Tmux.NewWindow(id, env, []string{a.o.SelfPath, "launch", id})
	if err != nil {
		_ = launch.Remove(a.o.Paths.LaunchDir, id)
		return terminals.Terminal{}, err
	}
	a.reg.Add(terminals.Terminal{ID: id, Name: spec.Name, Preset: spec.Preset, Cwd: spec.Cwd, Window: w.ID, Pane: w.Pane})
	a.emitTerminals()
	t, _ := a.reg.Get(id)
	return t, nil
}

func (a *App) NewTerminal(preset string) (terminals.Terminal, error) {
	if preset == "" {
		return a.open(launch.Spec{Name: "shell", Cwd: a.o.Home})
	}
	p, ok := a.o.Config.Preset(preset)
	if !ok {
		return terminals.Terminal{}, fmt.Errorf("preset desconhecido: %q", preset)
	}
	return a.open(launch.Spec{Name: p.Name, Preset: p.Name, Cwd: a.o.Home, Command: p.Command})
}

func (a *App) Resume(sessionID string) (terminals.Terminal, error) {
	c, ok := a.o.Store.Get(sessionID)
	if !ok {
		return terminals.Terminal{}, fmt.Errorf("conversa não encontrada: %q", sessionID)
	}
	command, err := resume.Command(sessionID)
	if err != nil {
		return terminals.Terminal{}, err
	}
	name := c.Title
	if name == "" {
		name = "retomada"
	}
	t, err := a.open(launch.Spec{Name: name, Preset: c.Preset, Cwd: c.Cwd, Command: command})
	if err != nil {
		return t, err
	}
	if err := a.o.Store.Remove(sessionID); err != nil {
		a.o.Logf("forget %s: %v", sessionID, err)
	}
	a.emitConversations()
	return t, nil
}

func (a *App) get(id string) (terminals.Terminal, error) {
	t, ok := a.reg.Get(id)
	if !ok {
		return t, fmt.Errorf("terminal desconhecido: %q", id)
	}
	return t, nil
}

func (a *App) Write(id, data string) error {
	t, err := a.get(id)
	if err != nil {
		return err
	}
	return a.o.Tmux.SendKeys(t.Pane, []byte(data))
}

func (a *App) Resize(id string, cols, rows int) error {
	t, err := a.get(id)
	if err != nil {
		return err
	}
	return a.o.Tmux.Resize(t.Window, cols, rows)
}

func (a *App) Snapshot(id string) (string, error) {
	t, err := a.get(id)
	if err != nil {
		return "", err
	}
	return a.o.Tmux.Capture(t.Pane)
}

// Reorder saves the order the user dragged the terminals into.
func (a *App) Reorder(ids []string) error {
	for _, t := range a.reg.Reorder(ids) {
		spec, err := launch.Read(a.o.Paths.LaunchDir, t.ID)
		if err != nil {
			continue
		}
		spec.Order = t.Order
		if err := launch.Write(a.o.Paths.LaunchDir, t.ID, spec); err != nil {
			a.o.Logf("reorder %s: %v", t.ID, err)
		}
	}
	a.emitTerminals()
	return nil
}

// HandleUsage keeps the plan usage the status line last reported and tells the clients when
// the numbers change.
func (a *App) HandleUsage(u hooks.Usage) {
	u.UpdatedAt = a.o.Now()
	a.mu.Lock()
	changed := !sameWindow(a.usage.FiveHour, u.FiveHour) || !sameWindow(a.usage.SevenDay, u.SevenDay)
	a.usage = u
	a.mu.Unlock()
	if changed {
		a.o.Emit("usage", u)
	}
}

func (a *App) Usage() hooks.Usage {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.usage
}

func sameWindow(x, y *hooks.Window) bool {
	if x == nil || y == nil {
		return x == y
	}
	return *x == *y
}

func (a *App) Rename(id, name string) error {
	if _, ok := a.reg.Rename(id, name); !ok {
		return fmt.Errorf("nome inválido ou terminal desconhecido")
	}
	if spec, err := launch.Read(a.o.Paths.LaunchDir, id); err == nil {
		spec.Name = strings.TrimSpace(name)
		if err := launch.Write(a.o.Paths.LaunchDir, id, spec); err != nil {
			a.o.Logf("rename %s: %v", id, err)
		}
	}
	a.emitTerminals()
	return nil
}

func (a *App) archive(t terminals.Terminal) {
	c, ok := t.Conversation(a.o.Now())
	if !ok {
		return
	}
	if err := a.o.Store.Add(c); err != nil {
		a.o.Logf("archive %s: %v", t.ID, err)
		return
	}
	a.emitConversations()
}

func (a *App) forget(t terminals.Terminal) {
	a.reg.Remove(t.ID)
	_ = launch.Remove(a.o.Paths.LaunchDir, t.ID)
	a.emitTerminals()
}

func (a *App) Close(id string) error {
	t, err := a.get(id)
	if err != nil {
		return err
	}
	a.archive(t)
	a.forget(t)
	if err := a.o.Tmux.KillWindow(t.Window); err != nil {
		a.o.Logf("kill %s: %v", t.Window, err)
	}
	return nil
}

func (a *App) HandleWindowClosed(window string) {
	t, ok := a.reg.ByWindow(window)
	if !ok {
		return
	}
	a.archive(t)
	a.forget(t)
}

func (a *App) Forget(sessionID string) error {
	err := a.o.Store.Remove(sessionID)
	a.emitConversations()
	return err
}

func (a *App) List() []terminals.Terminal           { return a.reg.List() }
func (a *App) Conversations() []resume.Conversation { return a.o.Store.List() }

func (a *App) Presets() []config.Preset {
	return append([]config.Preset{}, a.o.Config.Presets...)
}

func (a *App) Sound() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.sound
}

func (a *App) SetSound(on bool) {
	a.mu.Lock()
	a.sound = on
	a.mu.Unlock()
}

func (a *App) SetFocused(focused bool) {
	a.mu.Lock()
	a.focused = focused
	a.mu.Unlock()
}

func (a *App) Quit() error {
	for _, t := range a.reg.List() {
		a.archive(t)
		a.forget(t)
	}
	return a.o.Tmux.KillServer()
}

func (a *App) HandleHook(ev hooks.Event) {
	ch, ok := a.reg.Apply(ev)
	if !ok {
		a.o.Logf("hook ignored: %s for %s", ev.Name, ev.Terminal)
		return
	}
	if ch.Ended != nil {
		if err := a.o.Store.Add(*ch.Ended); err != nil {
			a.o.Logf("archive: %v", err)
		}
		a.emitConversations()
	}
	a.emitTerminals()
	if ch.Attention {
		a.alert(ch.Terminal)
	}
}

func (a *App) alert(t terminals.Terminal) {
	a.mu.Lock()
	sound, focused := a.sound, a.focused
	a.mu.Unlock()
	if sound {
		a.o.Emit("bark", t.ID)
	}
	if focused {
		return
	}
	body := "terminou"
	if t.State == terminals.Waiting {
		body = "esperando você"
		if t.Ask != "" {
			body = t.Ask
		}
	}
	go func() {
		if err := a.o.Notifier.Show(t.Name, body); err != nil {
			a.o.Logf("toast: %v", err)
		}
	}()
}

func (a *App) HandleOutput(pane string, data []byte) {
	t, ok := a.reg.ByPane(pane)
	if !ok {
		return
	}
	a.o.Emit("output", OutputEvent{ID: t.ID, Data: base64.StdEncoding.EncodeToString(data)})
}

func (a *App) emitTerminals()     { a.o.Emit("terminals", a.reg.List()) }
func (a *App) emitConversations() { a.o.Emit("conversations", a.o.Store.List()) }
