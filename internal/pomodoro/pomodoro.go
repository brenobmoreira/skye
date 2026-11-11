// Package pomodoro runs focus and break phases. A focus ends into a break on its own; a break ends
// into idle, and the next focus waits for start.
package pomodoro

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

type Phase string

const (
	Idle  Phase = "idle"
	Focus Phase = "focus"
	Break Phase = "break"
	Long  Phase = "long"
)

type State struct {
	Phase       Phase `json:"phase"`
	Paused      bool  `json:"paused"`
	RemainingMs int64 `json:"remainingMs"`
	TotalMs     int64 `json:"totalMs"`
	Count       int   `json:"count"`
	// Stage is how many focuses of the current cycle are done; the long break starts it over.
	Stage    int      `json:"stage"`
	Settings Settings `json:"settings"`
}

// Ended is a phase that ran out on its own, and the state it left.
type Ended struct {
	Phase Phase
	Next  State
}

// After schedules f after d and returns a stop, like time.AfterFunc.
type After func(d time.Duration, f func()) (stop func() bool)

type Timer struct {
	mu        sync.Mutex
	settings  Settings
	now       func() time.Time
	after     After
	onChange  func(State)
	onEnd     func(Ended)
	phase     Phase
	paused    bool
	endsAt    time.Time
	remaining time.Duration
	total     time.Duration
	count     int
	stage     int
	day       string
	stop      func() bool
	gen       int
}

func New(s Settings, now func() time.Time, after After, onChange func(State), onEnd func(Ended)) *Timer {
	return &Timer{settings: s, now: now, after: after, onChange: onChange, onEnd: onEnd, phase: Idle}
}

func (t *Timer) today() string { return t.now().Format("2006-01-02") }

func (t *Timer) state() State {
	st := State{Phase: t.phase, Paused: t.paused, Settings: t.settings, Stage: t.stage}
	if t.day == t.today() {
		st.Count = t.count
	}
	if t.phase != Idle {
		left := t.remaining
		if !t.paused {
			left = max(t.endsAt.Sub(t.now()), 0)
		}
		st.RemainingMs = left.Milliseconds()
		st.TotalMs = t.total.Milliseconds()
	}
	return st
}

func (t *Timer) State() State {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.state()
}

func (t *Timer) cancel() {
	t.gen++
	if t.stop != nil {
		t.stop()
		t.stop = nil
	}
}

// run starts phase p with d left of a phase that lasts total.
func (t *Timer) run(p Phase, d, total time.Duration) {
	t.cancel()
	t.phase, t.paused, t.total = p, false, total
	t.endsAt = t.now().Add(d)
	gen := t.gen
	t.stop = t.after(d, func() { t.fire(gen) })
}

func (t *Timer) idle() {
	t.cancel()
	t.phase, t.paused, t.remaining, t.total = Idle, false, 0, 0
}

func minutes(m int) time.Duration { return time.Duration(m) * time.Minute }

// runBreak starts the break that follows the focuses done: the long one closes the cycle.
func (t *Timer) runBreak() {
	if t.stage >= t.settings.Every {
		t.stage = 0
		t.run(Long, minutes(t.settings.Long), minutes(t.settings.Long))
		return
	}
	t.run(Break, minutes(t.settings.Break), minutes(t.settings.Break))
}

func (t *Timer) fire(gen int) {
	t.mu.Lock()
	if gen != t.gen || t.phase == Idle || t.paused {
		t.mu.Unlock()
		return
	}
	t.stop = nil
	ended := t.phase
	if ended == Focus {
		// The focus counts for the day it ends on.
		if today := t.today(); t.day != today {
			t.day, t.count = today, 0
		}
		t.count++
		t.stage++
		t.runBreak()
	} else {
		t.idle()
	}
	st := t.state()
	t.mu.Unlock()
	t.onEnd(Ended{Phase: ended, Next: st})
	t.onChange(st)
}

// Do runs one of start, pause, resume, skip or reset.
func (t *Timer) Do(action string) error {
	t.mu.Lock()
	switch action {
	case "start":
		if t.phase != Idle {
			t.mu.Unlock()
			return errors.New("já está rodando")
		}
		t.run(Focus, minutes(t.settings.Focus), minutes(t.settings.Focus))
	case "pause":
		if t.phase == Idle || t.paused {
			t.mu.Unlock()
			return errors.New("nada rodando para pausar")
		}
		t.remaining = max(t.endsAt.Sub(t.now()), 0)
		t.cancel()
		t.paused = true
	case "resume":
		if !t.paused {
			t.mu.Unlock()
			return errors.New("nada pausado")
		}
		t.run(t.phase, t.remaining, t.total)
	case "skip":
		if t.phase == Idle {
			t.mu.Unlock()
			return errors.New("nada para pular")
		}
		t.idle()
	case "reset":
		t.idle()
		t.count, t.stage = 0, 0
	default:
		t.mu.Unlock()
		return fmt.Errorf("ação desconhecida: %q", action)
	}
	st := t.state()
	t.mu.Unlock()
	t.onChange(st)
	return nil
}

// Begin starts from a chosen point of the cycle: the focus of stage n, or the break that follows
// it (the long one after the last stage). Choosing does not change the day's count.
func (t *Timer) Begin(p Phase, n int) error {
	t.mu.Lock()
	if t.phase != Idle {
		t.mu.Unlock()
		return errors.New("já está rodando")
	}
	if n < 1 || n > t.settings.Every {
		t.mu.Unlock()
		return fmt.Errorf("estágio de 1 a %d", t.settings.Every)
	}
	switch p {
	case Focus:
		t.stage = n - 1
		t.run(Focus, minutes(t.settings.Focus), minutes(t.settings.Focus))
	case Break:
		t.stage = n
		t.runBreak()
	default:
		t.mu.Unlock()
		return fmt.Errorf("só dá para começar por foco ou pausa, não %q", p)
	}
	st := t.state()
	t.mu.Unlock()
	t.onChange(st)
	return nil
}

// SetSettings takes effect from the next phase; the one running keeps its length.
func (t *Timer) SetSettings(s Settings) error {
	if err := s.Validate(); err != nil {
		return err
	}
	t.mu.Lock()
	t.settings = s
	st := t.state()
	t.mu.Unlock()
	t.onChange(st)
	return nil
}
