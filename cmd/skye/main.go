package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"

	"github.com/brenobmoreira/skye/frontend"
	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/launch"
)

const usage = `skye — terminais do Claude Code no WSL

  skye                 abre a janela
  skye install-hooks   instala os hooks em ~/.claude/settings.json
  skye launch <id>     uso interno: roda o comando de um terminal`

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "launch":
			os.Exit(runLaunch(os.Args[2:]))
		case "install-hooks":
			os.Exit(runInstallHooks())
		case "-h", "--help", "help":
			fmt.Println(usage)
			return
		default:
			fmt.Fprintf(os.Stderr, "skye: comando desconhecido %q\n\n%s\n", os.Args[1], usage)
			os.Exit(2)
		}
	}
	runApp()
}

func runLaunch(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "uso: skye launch <id>")
		return 2
	}
	paths := config.DefaultPaths(os.Getenv)
	cfg, _ := config.Load(paths.ConfigFile)
	if err := launch.Exec(paths.LaunchDir, args[0], cfg.ResolveShell(os.Getenv)); err != nil {
		fmt.Fprintln(os.Stderr, "skye launch:", err)
		return 1
	}
	return 0
}

func runInstallHooks() int {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	path := filepath.Join(home, ".claude", "settings.json")
	backup, err := hooks.Install(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "skye install-hooks:", err)
		return 1
	}
	fmt.Println("hooks instalados em", path)
	if backup != "" {
		fmt.Println("backup do arquivo anterior:", backup)
	}
	fmt.Println("reinicie as sessões do claude para elas passarem a reportar")
	return 0
}

func runApp() {
	if hooks.RequestShow(config.DefaultPaths(os.Getenv).Socket) == nil {
		fmt.Println("a skye já está aberta")
		return
	}
	bridge := newBridge()
	assets, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = wails.Run(&options.App{
		Title:             "skye",
		Width:             1200,
		Height:            800,
		MinWidth:          700,
		MinHeight:         450,
		Frameless:         true,
		HideWindowOnClose: true,
		BackgroundColour:  &options.RGBA{R: 18, G: 18, B: 22, A: 255},
		AssetServer:       &assetserver.Options{Assets: assets},
		SingleInstanceLock: &options.SingleInstanceLock{
			UniqueId:               "io.github.brenobmoreira.skye",
			OnSecondInstanceLaunch: func(options.SecondInstanceData) { bridge.show() },
		},
		OnStartup:  bridge.startup,
		OnShutdown: bridge.shutdown,
		Bind:       []interface{}{bridge},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
