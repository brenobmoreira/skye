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
		AssetServer:      &assetserver.Options{Assets: assets},
		Windows:          &windows.Options{},
		Bind:             []interface{}{&Shell{}},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
