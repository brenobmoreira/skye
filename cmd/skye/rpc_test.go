package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/brenobmoreira/skye/internal/app"
	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/places"
	"github.com/brenobmoreira/skye/internal/procs"
	"github.com/brenobmoreira/skye/internal/resume"
	"github.com/brenobmoreira/skye/internal/terminals"
)

type fakeTarget struct {
	calls []string
}

func (f *fakeTarget) record(format string, args ...any) {
	f.calls = append(f.calls, fmt.Sprintf(format, args...))
}

func (f *fakeTarget) List() []terminals.Terminal {
	f.record("List")
	return []terminals.Terminal{{ID: "t1"}}
}
func (f *fakeTarget) Conversations() []resume.Conversation {
	f.record("Conversations")
	return []resume.Conversation{{SessionID: "s1"}}
}
func (f *fakeTarget) Presets() []config.Preset {
	f.record("Presets")
	return []config.Preset{{Name: "blog"}}
}
func (f *fakeTarget) NewTerminal(preset string) (terminals.Terminal, error) {
	f.record("NewTerminal(%s)", preset)
	return terminals.Terminal{ID: "t2"}, nil
}
func (f *fakeTarget) Resume(sessionID string) (terminals.Terminal, error) {
	f.record("Resume(%s)", sessionID)
	return terminals.Terminal{ID: "t3"}, nil
}
func (f *fakeTarget) Write(id, data string) error {
	f.record("Write(%s,%s)", id, data)
	return nil
}
func (f *fakeTarget) Resize(id string, cols, rows int) error {
	f.record("Resize(%s,%d,%d)", id, cols, rows)
	return nil
}
func (f *fakeTarget) Snapshot(id string) (string, error) {
	f.record("Snapshot(%s)", id)
	return "tela", nil
}
func (f *fakeTarget) Monitor() (app.MonitorReport, error) {
	f.record("Monitor")
	return app.MonitorReport{Machine: procs.Machine{MemAvailMB: 1234}}, nil
}

func (f *fakeTarget) Places() []places.Place {
	f.record("Places")
	return []places.Place{{Name: "LivIA", Path: "~/projects/ai_livia_copilot"}}
}

func (f *fakeTarget) AddPlace(name, path string) error {
	f.record("AddPlace(%s,%s)", name, path)
	return nil
}

func (f *fakeTarget) RemovePlace(name string) error {
	f.record("RemovePlace(%s)", name)
	return nil
}

func (f *fakeTarget) OpenPlace(name string) (terminals.Terminal, error) {
	f.record("OpenPlace(%s)", name)
	return terminals.Terminal{ID: "t5"}, nil
}

func (f *fakeTarget) Repos() []config.Repo {
	f.record("Repos")
	return []config.Repo{{Name: "livia", Path: "/secret"}}
}

func (f *fakeTarget) NewWorktree(repo, branch string) (terminals.Terminal, error) {
	f.record("NewWorktree(%s,%s)", repo, branch)
	return terminals.Terminal{ID: "t4"}, nil
}

func (f *fakeTarget) Usage() hooks.Usage {
	f.record("Usage")
	return hooks.Usage{FiveHour: &hooks.Window{UsedPct: 5}}
}
func (f *fakeTarget) Reorder(ids []string) error {
	f.record("Reorder(%s)", strings.Join(ids, ","))
	return nil
}
func (f *fakeTarget) Rename(id, name string) error {
	f.record("Rename(%s,%s)", id, name)
	return nil
}
func (f *fakeTarget) Close(id string) error {
	f.record("Close(%s)", id)
	return errors.New("fechado")
}
func (f *fakeTarget) Forget(sessionID string) error {
	f.record("Forget(%s)", sessionID)
	return nil
}
func (f *fakeTarget) Sound() bool {
	f.record("Sound")
	return true
}
func (f *fakeTarget) SetSound(on bool) { f.record("SetSound(%v)", on) }
func (f *fakeTarget) SetFocused(focused bool) {
	f.record("SetFocused(%v)", focused)
}
func (f *fakeTarget) Quit() error {
	f.record("Quit")
	return nil
}
func (f *fakeTarget) Problems() []string {
	f.record("Problems")
	return []string{"p"}
}

func raw(t *testing.T, args string) []json.RawMessage {
	t.Helper()
	var out []json.RawMessage
	if err := json.Unmarshal([]byte(args), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestDispatchCallsEachMethodWithDecodedArgs(t *testing.T) {
	cases := []struct {
		method, args, call, result string
		err                        bool
	}{
		{"List", `[]`, "List", `[{"id":"t1"`, false},
		{"Conversations", `[]`, "Conversations", `[{"sessionId":"s1"`, false},
		{"Presets", `[]`, "Presets", `[{"name":"blog"`, false},
		{"NewTerminal", `["blog"]`, "NewTerminal(blog)", `{"id":"t2"`, false},
		{"Resume", `["s9"]`, "Resume(s9)", `{"id":"t3"`, false},
		{"Write", `["t1","ls\r"]`, "Write(t1,ls\r)", `null`, false},
		{"Resize", `["t1",120,40]`, "Resize(t1,120,40)", `null`, false},
		{"Snapshot", `["t1"]`, "Snapshot(t1)", `"tela"`, false},
		{"Rename", `["t1","demo"]`, "Rename(t1,demo)", `null`, false},
		{"Reorder", `[["t2","t1"]]`, "Reorder(t2,t1)", `null`, false},
		{"Close", `["t1"]`, "Close(t1)", "", true},
		{"Forget", `["s1"]`, "Forget(s1)", `null`, false},
		{"Sound", `[]`, "Sound", `true`, false},
		{"SetSound", `[false]`, "SetSound(false)", `null`, false},
		{"SetFocused", `[true]`, "SetFocused(true)", `null`, false},
		{"Quit", `[]`, "Quit", `null`, false},
		{"Problems", `[]`, "Problems", `["p"]`, false},
		{"Usage", `[]`, "Usage", `{"fiveHour":{"usedPct":5`, false},
		{"Places", `[]`, "Places", `[{"name":"LivIA","path":"~/projects/ai_livia_copilot"}]`, false},
		{"AddPlace", `["LivIA","~/x"]`, "AddPlace(LivIA,~/x)", `null`, false},
		{"RemovePlace", `["LivIA"]`, "RemovePlace(LivIA)", `null`, false},
		{"OpenPlace", `["LivIA"]`, "OpenPlace(LivIA)", `{"id":"t5"`, false},
		{"Repos", `[]`, "Repos", `[{"name":"livia"}]`, false},
		{"NewWorktree", `["livia","feature/x"]`, "NewWorktree(livia,feature/x)", `{"id":"t4"`, false},
		{"Monitor", `[]`, "Monitor", `{"machine":{"memTotalMb":0,"memAvailMb":1234`, false},
	}
	for _, c := range cases {
		f := &fakeTarget{}
		result, err := dispatcher(f)(c.method, raw(t, c.args))
		if c.err != (err != nil) {
			t.Fatalf("%s: err = %v", c.method, err)
		}
		if !reflect.DeepEqual(f.calls, []string{c.call}) {
			t.Fatalf("%s: calls = %q, want %q", c.method, f.calls, c.call)
		}
		if c.err {
			continue
		}
		out, _ := json.Marshal(result)
		if !strings.HasPrefix(string(out), c.result) {
			t.Fatalf("%s: result = %s, want prefix %s", c.method, out, c.result)
		}
	}
}

func TestDispatchRejectsUnknownAndWindowOnlyMethods(t *testing.T) {
	for _, m := range []string{"Nope", "Hide", "ToggleMaximise", "list", ""} {
		f := &fakeTarget{}
		if _, err := dispatcher(f)(m, nil); err == nil {
			t.Fatalf("%q: expected error", m)
		}
		if len(f.calls) != 0 {
			t.Fatalf("%q: calls = %v", m, f.calls)
		}
	}
}

func TestDispatchRejectsWrongArgs(t *testing.T) {
	for _, c := range []struct{ method, args string }{
		{"List", `[1]`},
		{"NewTerminal", `[]`},
		{"NewTerminal", `[1]`},
		{"Write", `["t1"]`},
		{"Write", `["t1","a","b"]`},
		{"Resize", `["t1","120",40]`},
		{"Resize", `["t1",1.5,40]`},
		{"SetSound", `["sim"]`},
		{"SetFocused", `[]`},
	} {
		f := &fakeTarget{}
		if _, err := dispatcher(f)(c.method, raw(t, c.args)); err == nil {
			t.Fatalf("%s %s: expected error", c.method, c.args)
		}
		if len(f.calls) != 0 {
			t.Fatalf("%s %s: calls = %v", c.method, c.args, f.calls)
		}
	}
}
