package app

import (
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/notes"
	"github.com/brenobmoreira/skye/internal/pomodoro"
)

type focusRig struct {
	app      *App
	notif    *fakeNotifier
	mu       sync.Mutex
	now      time.Time
	events   []string
	barks    []string
	problems []string
	jobs     []func()
	durs     []time.Duration
	file     string
}

func newFocusRig(t *testing.T, sound bool) *focusRig {
	t.Helper()
	dir := t.TempDir()
	r := &focusRig{notif: &fakeNotifier{}, now: time.Date(2026, 11, 11, 10, 0, 0, 0, time.Local), file: filepath.Join(dir, "pomodoro.json")}
	now := func() time.Time { r.mu.Lock(); defer r.mu.Unlock(); return r.now }
	ids := 0
	store, err := notes.Open(filepath.Join(dir, "notes.json"), now, func() string { ids++; return "n" + string(rune('0'+ids)) })
	if err != nil {
		t.Fatal(err)
	}
	r.app = New(Options{
		Config:   config.Config{Sound: sound},
		Notifier: r.notif,
		Emit: func(ev string, data any) {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.events = append(r.events, ev)
			if ev == "bark" {
				r.barks = append(r.barks, data.(string))
			}
		},
		Logf:         t.Logf,
		Now:          now,
		Notes:        store,
		PomodoroFile: r.file,
		After: func(d time.Duration, f func()) func() bool {
			r.mu.Lock()
			defer r.mu.Unlock()
			r.jobs = append(r.jobs, f)
			r.durs = append(r.durs, d)
			return func() bool { return true }
		},
		Problem: func(msg string) { r.mu.Lock(); defer r.mu.Unlock(); r.problems = append(r.problems, msg) },
	})
	return r
}

func (r *focusRig) count(ev string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for _, e := range r.events {
		if e == ev {
			n++
		}
	}
	return n
}

func (r *focusRig) at(h, m, s int) time.Time {
	return time.Date(2026, 11, 11, h, m, s, 0, time.Local)
}

// fire advances the clock past the last scheduled phase and runs it.
func (r *focusRig) fire() {
	r.mu.Lock()
	f, d := r.jobs[len(r.jobs)-1], r.durs[len(r.durs)-1]
	r.now = r.now.Add(d)
	r.mu.Unlock()
	f()
}

func (r *focusRig) shown() []string {
	r.notif.mu.Lock()
	defer r.notif.mu.Unlock()
	return append([]string{}, r.notif.shown...)
}

func TestNoteChangesEmitNotes(t *testing.T) {
	r := newFocusRig(t, true)
	it, err := r.app.AddNote("a")
	if err != nil {
		t.Fatal(err)
	}
	steps := []func() error{
		func() error { _, err := r.app.EditNote(it.ID, notes.Edit{Text: "b"}); return err },
		func() error { return r.app.CheckNote(it.ID) },
		func() error { return r.app.UncheckNote(it.ID) },
		func() error { return r.app.DeleteNote(it.ID) },
	}
	for i, step := range steps {
		if err := step(); err != nil {
			t.Fatal(err)
		}
		if got := r.count("notes"); got != i+2 {
			t.Fatalf("after step %d notes events = %d", i, got)
		}
	}
	if _, err := r.app.AddNote(" "); err == nil {
		t.Fatal("blank note accepted")
	}
	if got := r.count("notes"); got != 5 {
		t.Fatalf("a failed change emitted: %d", got)
	}
}

func timed(t *testing.T, r *focusRig, text, date, hhmm string) notes.Item {
	t.Helper()
	it, err := r.app.AddNote(text)
	if err != nil {
		t.Fatal(err)
	}
	it, err = r.app.EditNote(it.ID, notes.Edit{Text: text, Date: date, Time: hhmm})
	if err != nil {
		t.Fatal(err)
	}
	return it
}

func TestAlarmsDoNotFireForTimesBeforeStart(t *testing.T) {
	r := newFocusRig(t, true)
	timed(t, r, "reunião", "2026-11-11", "09:00")
	r.app.CheckAlarms(r.at(10, 0, 0))
	r.app.CheckAlarms(r.at(10, 1, 0))
	time.Sleep(20 * time.Millisecond)
	if len(r.barks) != 0 || len(r.shown()) != 0 {
		t.Fatalf("barks %v shown %v", r.barks, r.shown())
	}
}

func TestAlarmFiresOnceAtItsTime(t *testing.T) {
	r := newFocusRig(t, true)
	it := timed(t, r, "revisar spec", "2026-11-11", "10:05")
	for _, at := range []time.Time{r.at(10, 0, 0), r.at(10, 4, 30), r.at(10, 5, 10), r.at(10, 6, 0)} {
		r.app.CheckAlarms(at)
	}
	waitFor(t, func() bool { return len(r.shown()) == 1 })
	if !slices.Equal(r.barks, []string{it.ID}) {
		t.Fatalf("barks %v", r.barks)
	}
	if got := r.shown()[0]; got != "skye|revisar spec" {
		t.Fatalf("shown %q", got)
	}

	f := newFocusRig(t, true)
	f.app.SetFocused(true)
	timed(t, f, "x", "2026-11-11", "10:05")
	f.app.CheckAlarms(f.at(10, 0, 0))
	f.app.CheckAlarms(f.at(10, 5, 0))
	time.Sleep(20 * time.Millisecond)
	if len(f.barks) != 1 || len(f.shown()) != 0 {
		t.Fatalf("focused: barks %v shown %v", f.barks, f.shown())
	}

	d := newFocusRig(t, true)
	done := timed(t, d, "x", "2026-11-11", "10:05")
	d.app.CheckNote(done.ID)
	d.app.CheckAlarms(d.at(10, 0, 0))
	d.app.CheckAlarms(d.at(10, 5, 0))
	time.Sleep(20 * time.Millisecond)
	if len(d.barks) != 0 || len(d.shown()) != 0 {
		t.Fatalf("done: barks %v shown %v", d.barks, d.shown())
	}
}

func TestPomodoroEndNotifiesWithoutBarking(t *testing.T) {
	r := newFocusRig(t, true)
	r.app.SetFocused(true)
	if err := r.app.PomodoroAction("start"); err != nil {
		t.Fatal(err)
	}
	r.fire()
	waitFor(t, func() bool { return len(r.shown()) == 1 })
	if got := r.shown()[0]; got != "skye|pausa de 5 min" {
		t.Fatalf("shown %q", got)
	}
	r.fire()
	waitFor(t, func() bool { return len(r.shown()) == 2 })
	if got := r.shown()[1]; got != "skye|hora de focar" {
		t.Fatalf("shown %q", got)
	}
	for range 3 {
		r.app.PomodoroAction("start")
		r.fire()
		r.app.PomodoroAction("skip")
	}
	waitFor(t, func() bool { return len(r.shown()) == 5 })
	// Toasts go out on their own goroutines, so only the set is fixed, not the order.
	if !slices.Contains(r.shown(), "skye|pausa longa de 15 min") {
		t.Fatalf("shown %q", r.shown())
	}
	if len(r.barks) != 0 {
		t.Fatalf("pomodoro barked: %v", r.barks)
	}
}

func TestPomodoroActionsEmitState(t *testing.T) {
	r := newFocusRig(t, true)
	if err := r.app.PomodoroAction("start"); err != nil {
		t.Fatal(err)
	}
	if r.count("pomodoro") != 1 || r.app.Pomodoro().Phase != pomodoro.Focus {
		t.Fatalf("events %v state %+v", r.events, r.app.Pomodoro())
	}
	if err := r.app.PomodoroAction("nap"); err == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestSetPomodoroSettingsSaves(t *testing.T) {
	r := newFocusRig(t, true)
	want := pomodoro.Settings{Focus: 50, Break: 10, Long: 20, Every: 3}
	if err := r.app.SetPomodoroSettings(want); err != nil {
		t.Fatal(err)
	}
	if got, err := pomodoro.LoadSettings(r.file); err != nil || got != want {
		t.Fatalf("saved %+v %v", got, err)
	}
	if r.app.Pomodoro().Settings != want {
		t.Fatalf("not applied: %+v", r.app.Pomodoro().Settings)
	}
	if err := r.app.SetPomodoroSettings(pomodoro.Settings{}); err == nil {
		t.Fatal("invalid settings accepted")
	}
	if got, _ := pomodoro.LoadSettings(r.file); got != want {
		t.Fatalf("file changed by invalid settings: %+v", got)
	}
}
