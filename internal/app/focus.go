package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/brenobmoreira/skye/internal/notes"
	"github.com/brenobmoreira/skye/internal/pomodoro"
)

var errNoNotes = errors.New("anotações indisponíveis")

// maxAlarmGap is the longest stretch between two checks that still rings what fell in it.
const maxAlarmGap = 5 * time.Minute

func (a *App) startPomodoro() {
	settings := pomodoro.Default
	if a.o.PomodoroFile != "" {
		s, err := pomodoro.LoadSettings(a.o.PomodoroFile)
		if err != nil {
			a.o.Problem(fmt.Sprintf("tempos do pomodoro ilegíveis, usando 25/5/15/4: %v", err))
		}
		settings = s
	}
	a.timer = pomodoro.New(settings, a.o.Now, a.o.After, a.pomodoroChanged, a.pomodoroEnded)
}

func (a *App) pomodoroChanged(s pomodoro.State) { a.o.Emit("pomodoro", s) }

// pomodoroEnded only notifies: no bark, and even with the window focused, since the toast is the
// only signal the phase is over.
func (a *App) pomodoroEnded(e pomodoro.Ended) {
	body := "hora de focar"
	if e.Phase == pomodoro.Focus {
		if e.Next.Phase == pomodoro.Long {
			body = fmt.Sprintf("pausa longa de %d min", e.Next.Settings.Long)
		} else {
			body = fmt.Sprintf("pausa de %d min", e.Next.Settings.Break)
		}
	}
	a.toast("skye", body)
	// Tells the windows a phase ran out on its own, so the closed panel's pill can pulse.
	a.o.Emit("pomodoroEnded", e.Phase)
}

func (a *App) toast(title, body string) {
	go func() {
		if err := a.o.Notifier.Show(title, body); err != nil {
			a.o.Logf("toast: %v", err)
		}
	}()
}

func (a *App) Pomodoro() pomodoro.State { return a.timer.State() }

func (a *App) PomodoroAction(action string) error { return a.timer.Do(action) }

// PomodoroBegin starts at a chosen stage: phase is "focus" or "break".
func (a *App) PomodoroBegin(phase string, stage int) error {
	return a.timer.Begin(pomodoro.Phase(phase), stage)
}

func (a *App) SetPomodoroSettings(s pomodoro.Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if a.o.PomodoroFile != "" {
		if err := pomodoro.SaveSettings(a.o.PomodoroFile, s); err != nil {
			return err
		}
	}
	return a.timer.SetSettings(s)
}

func (a *App) Notes() (notes.Notes, error) {
	if a.o.Notes == nil {
		return notes.Notes{Items: []notes.Item{}, Done: []notes.Done{}}, errNoNotes
	}
	n, err := a.o.Notes.List()
	a.reportNoteProblems()
	return n, err
}

func (a *App) reportNoteProblems() {
	for _, p := range a.o.Notes.Problems() {
		a.o.Problem(p)
	}
}

// changed reports files moved aside and, when the change went through, sends the list to every
// open window.
func (a *App) changed(err error) error {
	a.reportNoteProblems()
	if err != nil {
		return err
	}
	if n, err := a.o.Notes.List(); err == nil {
		a.o.Emit("notes", n)
	}
	return nil
}

func (a *App) AddNote(text string) (notes.Item, error) {
	if a.o.Notes == nil {
		return notes.Item{}, errNoNotes
	}
	it, err := a.o.Notes.Add(text)
	return it, a.changed(err)
}

func (a *App) EditNote(id string, e notes.Edit) (notes.Item, error) {
	if a.o.Notes == nil {
		return notes.Item{}, errNoNotes
	}
	it, err := a.o.Notes.Edit(id, e)
	return it, a.changed(err)
}

func (a *App) PinNote(id string, pinned bool) error {
	if a.o.Notes == nil {
		return errNoNotes
	}
	return a.changed(a.o.Notes.Pin(id, pinned))
}

func (a *App) MoveNote(id, to string) error {
	if a.o.Notes == nil {
		return errNoNotes
	}
	return a.changed(a.o.Notes.Move(id, to))
}

func (a *App) CheckNote(id string) error {
	if a.o.Notes == nil {
		return errNoNotes
	}
	return a.changed(a.o.Notes.Check(id))
}

func (a *App) UncheckNote(id string) error {
	if a.o.Notes == nil {
		return errNoNotes
	}
	return a.changed(a.o.Notes.Uncheck(id))
}

func (a *App) DeleteNote(id string) error {
	if a.o.Notes == nil {
		return errNoNotes
	}
	return a.changed(a.o.Notes.Delete(id))
}

// CheckAlarms rings the notes whose date and time fall after the previous check and up to now.
// The first check only marks where skye started, so times missed while it was down stay quiet.
func (a *App) CheckAlarms(now time.Time) {
	a.mu.Lock()
	from := a.alarmsFrom
	a.alarmsFrom = now
	sound, focused := a.sound, a.focused
	a.mu.Unlock()
	// A gap far longer than a tick means the machine slept or the clock jumped: like a start,
	// what fell in it stays quiet.
	if from.IsZero() || now.Sub(from) > maxAlarmGap || a.o.Notes == nil {
		return
	}
	n, err := a.o.Notes.List()
	if err != nil {
		a.o.Logf("alarms: %v", err)
		return
	}
	for _, it := range n.Items {
		if it.Date == "" || it.Time == "" || !it.DoneAt.IsZero() {
			continue
		}
		at, err := time.ParseInLocation("2006-01-02 15:04", it.Date+" "+it.Time, now.Location())
		if err != nil || !at.After(from) || at.After(now) {
			continue
		}
		if sound {
			a.o.Emit("bark", it.ID)
		}
		if !focused {
			a.toast("skye", it.Text)
		}
	}
}

// WatchAlarms checks the notes every interval until stop closes (nil runs forever).
func (a *App) WatchAlarms(every time.Duration, stop <-chan struct{}) {
	a.CheckAlarms(a.o.Now())
	tick := time.NewTicker(every)
	defer tick.Stop()
	for {
		select {
		case <-stop:
			return
		case <-tick.C:
			a.CheckAlarms(a.o.Now())
		}
	}
}
