package main

import (
	"fmt"
	"reflect"
	"testing"
)

type fakeHost struct {
	calls  []string
	events []string
}

func (h *fakeHost) emit(event string, data any) {
	h.events = append(h.events, fmt.Sprintf("%s=%v", event, data))
}
func (h *fakeHost) logf(format string, args ...any) {}
func (h *fakeHost) logError(msg string)             { h.calls = append(h.calls, "logError:"+msg) }
func (h *fakeHost) show()                           { h.calls = append(h.calls, "show") }
func (h *fakeHost) hide()                           { h.calls = append(h.calls, "hide") }
func (h *fakeHost) toggleMaximise()                 { h.calls = append(h.calls, "toggleMaximise") }
func (h *fakeHost) quit()                           { h.calls = append(h.calls, "quit") }

func TestBridgeDelegatesWindowActionsToHost(t *testing.T) {
	h := &fakeHost{}
	b := newBridge(h)
	b.Hide()
	b.ToggleMaximise()
	if err := b.Quit(); err != nil {
		t.Fatal(err)
	}
	if want := []string{"hide", "toggleMaximise", "quit"}; !reflect.DeepEqual(h.calls, want) {
		t.Fatalf("calls = %v, want %v", h.calls, want)
	}
}

func TestBridgeProblemLogsAndEmits(t *testing.T) {
	h := &fakeHost{}
	b := newBridge(h)
	b.problem("tmux %s", "sumiu")
	if !reflect.DeepEqual(h.calls, []string{"logError:tmux sumiu"}) {
		t.Fatalf("calls = %v", h.calls)
	}
	if !reflect.DeepEqual(h.events, []string{"problems=[tmux sumiu]"}) {
		t.Fatalf("events = %v", h.events)
	}
	if got := b.Problems(); !reflect.DeepEqual(got, []string{"tmux sumiu"}) {
		t.Fatalf("problems = %v", got)
	}
}
