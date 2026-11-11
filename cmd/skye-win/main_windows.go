package main

import (
	"fmt"
	"io/fs"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"github.com/brenobmoreira/skye/frontend"
	"github.com/brenobmoreira/skye/internal/web"
	"github.com/brenobmoreira/skye/internal/winshell"
)

func main() {
	assets, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	err = wails.Run(&options.App{
		Title:            "skye",
		Width:            1200,
		Height:           800,
		MinWidth:         700,
		MinHeight:        450,
		Frameless:        true,
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 22, A: 255},
		// The window shows what the skye web in WSL serves, so updating skye there updates the
		// window; the built-in copy only stands in while that server is down.
		AssetServer: &assetserver.Options{Handler: &winshell.Proxy{
			Port:     winshell.Port(os.Getenv),
			Cookie:   web.CookieName,
			Token:    func() (string, error) { return winshell.ReadToken(run) },
			Probe:    winshell.Probe,
			Fallback: assets,
		}},
		Windows: &windows.Options{},
		Bind:    []interface{}{&Shell{}},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
