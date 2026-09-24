package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/brenobmoreira/skye/frontend"
	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/web"
)

type webHost struct {
	server *web.Server
	stop   func()
}

func (h *webHost) emit(event string, data any) { h.server.Broadcast(event, data) }

func (h *webHost) logf(format string, args ...any) { log.Printf(format, args...) }

func (h *webHost) logError(msg string) { log.Print(msg) }

func (h *webHost) show() {}

func (h *webHost) hide() {}

func (h *webHost) toggleMaximise() {}

func (h *webHost) quit() { h.stop() }

func runWeb(args []string) int {
	openBrowser := true
	for _, a := range args {
		if a != "--no-open" {
			fmt.Fprintln(os.Stderr, "uso: skye web [--no-open]")
			return 2
		}
		openBrowser = false
	}
	paths := config.DefaultPaths(os.Getenv)
	if hooks.RequestShow(paths.Socket) == nil {
		fmt.Println("a skye já está aberta")
		return 0
	}
	cfg, err := config.Load(paths.ConfigFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skye web: config inválida, usando padrões:", err)
	}
	token, err := web.LoadOrCreateToken(paths.WebToken)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skye web: token:", err)
		return 1
	}
	assets, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	h := &webHost{}
	bridge := newBridge(h)
	server, err := web.Listen(cfg.WebPort, web.Options{
		Token:    token,
		Assets:   assets,
		Dispatch: dispatcher(bridge),
		Logf:     log.Printf,

		OnLastClientGone: func() { bridge.SetFocused(false) },
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "skye web: não consegui abrir a porta %d (web_port no config.toml): %v\n", cfg.WebPort, err)
		return 1
	}
	h.server = server
	h.stop = func() { _ = server.Close() }
	if err := bridge.boot(); errors.Is(err, errOtherSkye) {
		_ = server.Close()
		fmt.Fprintln(os.Stderr, err)
		return 1
	} else if err != nil {
		bridge.problem("%v", err)
	}
	defer bridge.shutdown()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		_ = server.Close()
	}()
	url := server.URL()
	fmt.Println("skye no navegador:", url)
	if openBrowser {
		launchBrowser(url)
	}
	if err := server.Serve(); err != nil {
		fmt.Fprintln(os.Stderr, "skye web:", err)
		return 1
	}
	return 0
}

func launchBrowser(url string) {
	cmd := exec.Command("explorer.exe", url)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "não consegui abrir o navegador; abra o endereço acima:", err)
		return
	}
	go func() { _ = cmd.Wait() }()
}
