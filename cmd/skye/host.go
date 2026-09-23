package main

import (
	"context"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type host interface {
	emit(event string, data any)
	logf(format string, args ...any)
	logError(msg string)
	show()
	hide()
	toggleMaximise()
	quit()
}

type wailsHost struct {
	ctx context.Context
}

func (h *wailsHost) emit(event string, data any) { runtime.EventsEmit(h.ctx, event, data) }

func (h *wailsHost) logf(format string, args ...any) { runtime.LogInfof(h.ctx, format, args...) }

func (h *wailsHost) logError(msg string) { runtime.LogError(h.ctx, msg) }

func (h *wailsHost) show() {
	runtime.WindowUnminimise(h.ctx)
	runtime.WindowShow(h.ctx)
}

func (h *wailsHost) hide() { runtime.WindowHide(h.ctx) }

func (h *wailsHost) toggleMaximise() {
	if runtime.WindowIsFullscreen(h.ctx) {
		runtime.WindowUnfullscreen(h.ctx)
		return
	}
	if !runtime.WindowIsMaximised(h.ctx) {
		runtime.WindowMaximise(h.ctx)
		return
	}
	runtime.WindowUnmaximise(h.ctx)
	runtime.WindowSetSize(h.ctx, windowWidth, windowHeight)
	runtime.WindowCenter(h.ctx)
}

func (h *wailsHost) quit() { runtime.Quit(h.ctx) }
