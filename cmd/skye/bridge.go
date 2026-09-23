package main

import (
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/brenobmoreira/skye/internal/app"
	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/notify"
	"github.com/brenobmoreira/skye/internal/resume"
	"github.com/brenobmoreira/skye/internal/terminals"
	"github.com/brenobmoreira/skye/internal/tmuxctl"
)

//go:embed tmux.conf
var defaultTmuxConf []byte

const tmuxSocket = "skye"

var errNotReady = errors.New("a skye não conseguiu iniciar; veja o aviso no topo da janela")

type Bridge struct {
	ctx      context.Context
	mu       sync.Mutex
	app      *app.App
	client   *tmuxctl.Client
	server   *hooks.Server
	problems []string
}

func newBridge() *Bridge {
	return &Bridge{problems: []string{}}
}

func (b *Bridge) problem(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	b.mu.Lock()
	b.problems = append(b.problems, msg)
	list := append([]string{}, b.problems...)
	b.mu.Unlock()
	runtime.LogError(b.ctx, msg)
	runtime.EventsEmit(b.ctx, "problems", list)
}

func (b *Bridge) startup(ctx context.Context) {
	b.ctx = ctx
	if err := b.boot(); err != nil {
		b.problem("%v", err)
	}
}

func ensureFile(path string, content []byte) error {
	if _, err := os.Stat(path); err == nil {
		return nil
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o600)
}

func newID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return "t" + hex.EncodeToString(b)
}

func (b *Bridge) boot() error {
	paths := config.DefaultPaths(os.Getenv)
	cfg, err := config.Load(paths.ConfigFile)
	if err != nil {
		b.problem("config inválida, usando padrões: %v", err)
	}
	if err := tmuxctl.CheckVersion(); err != nil {
		return err
	}
	if err := ensureFile(paths.TmuxConf, defaultTmuxConf); err != nil {
		return err
	}
	if err := tmuxctl.EnsureServer(tmuxSocket, paths.TmuxConf); err != nil {
		return err
	}
	client, err := tmuxctl.Start(tmuxSocket)
	if err != nil {
		return err
	}
	store, err := resume.Open(paths.Conversations)
	if err != nil {
		b.problem("conversas encerradas ilegíveis, começando vazio: %v", err)
	}
	self, err := os.Executable()
	if err != nil {
		return err
	}
	home, _ := os.UserHomeDir()
	a := app.New(app.Options{
		Config:   cfg,
		Paths:    paths,
		Tmux:     client,
		Store:    store,
		Notifier: notify.NewToaster(),
		Emit:     func(event string, data any) { runtime.EventsEmit(b.ctx, event, data) },
		Logf:     func(format string, args ...any) { runtime.LogInfof(b.ctx, format, args...) },
		SelfPath: self,
		Home:     home,
		NewID:    newID,
		Now:      time.Now,
	})
	server, err := hooks.Listen(paths.Socket, a.HandleHook, func(format string, args ...any) { runtime.LogInfof(b.ctx, format, args...) })
	if errors.Is(err, hooks.ErrInUse) {
		return errors.New("outra skye já está rodando")
	}
	if err != nil {
		return err
	}
	if err := a.Recover(); err != nil {
		b.problem("não consegui reencontrar os terminais: %v", err)
	}
	b.mu.Lock()
	b.app, b.client, b.server = a, client, server
	b.mu.Unlock()
	go b.pump(client, a)
	return nil
}

func (b *Bridge) pump(c *tmuxctl.Client, a *app.App) {
	for {
		select {
		case o := <-c.Output:
			a.HandleOutput(o.Pane, o.Data)
		case w := <-c.Closed:
			a.HandleWindowClosed(w)
		case <-c.Exited:
			b.problem("o tmux da skye encerrou; reabra a skye")
			return
		}
	}
}

func (b *Bridge) shutdown(context.Context) {
	b.mu.Lock()
	server, client := b.server, b.client
	b.mu.Unlock()
	if server != nil {
		_ = server.Close()
	}
	if client != nil {
		_ = client.Close()
	}
}

func (b *Bridge) show() {
	runtime.WindowUnminimise(b.ctx)
	runtime.WindowShow(b.ctx)
}

func (b *Bridge) ready() (*app.App, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.app == nil {
		return nil, errNotReady
	}
	return b.app, nil
}

func (b *Bridge) Problems() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string{}, b.problems...)
}

func (b *Bridge) List() []terminals.Terminal {
	if a, err := b.ready(); err == nil {
		return a.List()
	}
	return []terminals.Terminal{}
}

func (b *Bridge) Conversations() []resume.Conversation {
	if a, err := b.ready(); err == nil {
		return a.Conversations()
	}
	return []resume.Conversation{}
}

func (b *Bridge) Presets() []config.Preset {
	if a, err := b.ready(); err == nil {
		return a.Presets()
	}
	return []config.Preset{}
}

func (b *Bridge) NewTerminal(preset string) (terminals.Terminal, error) {
	a, err := b.ready()
	if err != nil {
		return terminals.Terminal{}, err
	}
	return a.NewTerminal(preset)
}

func (b *Bridge) Resume(sessionID string) (terminals.Terminal, error) {
	a, err := b.ready()
	if err != nil {
		return terminals.Terminal{}, err
	}
	return a.Resume(sessionID)
}

func (b *Bridge) Write(id, data string) error {
	a, err := b.ready()
	if err != nil {
		return err
	}
	return a.Write(id, data)
}

func (b *Bridge) Paste(id, text string) error {
	a, err := b.ready()
	if err != nil {
		return err
	}
	return a.Paste(id, text)
}

func (b *Bridge) Resize(id string, cols, rows int) error {
	a, err := b.ready()
	if err != nil {
		return err
	}
	return a.Resize(id, cols, rows)
}

func (b *Bridge) Snapshot(id string) (string, error) {
	a, err := b.ready()
	if err != nil {
		return "", err
	}
	return a.Snapshot(id)
}

func (b *Bridge) Rename(id, name string) error {
	a, err := b.ready()
	if err != nil {
		return err
	}
	return a.Rename(id, name)
}

func (b *Bridge) Close(id string) error {
	a, err := b.ready()
	if err != nil {
		return err
	}
	return a.Close(id)
}

func (b *Bridge) Forget(sessionID string) error {
	a, err := b.ready()
	if err != nil {
		return err
	}
	return a.Forget(sessionID)
}

func (b *Bridge) Sound() bool {
	if a, err := b.ready(); err == nil {
		return a.Sound()
	}
	return false
}

func (b *Bridge) SetSound(on bool) {
	if a, err := b.ready(); err == nil {
		a.SetSound(on)
	}
}

func (b *Bridge) SetFocused(focused bool) {
	if a, err := b.ready(); err == nil {
		a.SetFocused(focused)
	}
}

func (b *Bridge) Hide() {
	runtime.WindowHide(b.ctx)
}

func (b *Bridge) Quit() error {
	if a, err := b.ready(); err == nil {
		if err := a.Quit(); err != nil {
			runtime.LogError(b.ctx, err.Error())
		}
	}
	runtime.Quit(b.ctx)
	return nil
}
