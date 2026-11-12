// Package pomodoro runs focus and break phases. Every phase ends into idle and the next one waits
// for start: after a focus, start runs the break that is due.
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
	Stage int `json:"stage"`
	// NextMs, set while idle, is the length the next phase starts with instead of the settings.
	NextMs int64 `json:"nextMs"`
	// Pending, set while idle after a focus ran out, is the break start runs: break or long.
	Pending  Phase    `json:"pending,omitempty"`
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
	next      time.Duration
	breakDue  bool
	day       string
	stop      func() bool
	gen       int
}

func New(s Settings, now func() time.Time, after After, onChange func(State), onEnd func(Ended)) *Timer {
	return &Timer{settings: s, now: now, after: after, onChange: onChange, onEnd: onEnd, phase: Idle}
}

func (t *Timer) today() string { return t.now().Format("2006-01-02") }

// checkEvery caps each wait for the end of a phase. Go timers wait on the monotonic clock, which
// stands still while Windows sleeps the WSL VM; a single wait for the whole phase would end it
// that much late, with the interface already at 0:00. Short waits check the wall clock instead.
const checkEvery = time.Second

// wall is now on the wall clock alone: Round(0) drops the monotonic reading, so the time left
// counts the sleep too.
func (t *Timer) wall() time.Time { return t.now().Round(0) }

// arm waits for the end of the running phase, a check at a time.
func (t *Timer) arm() {
	gen := t.gen
	wait := min(max(t.endsAt.Sub(t.wall()), 0), checkEvery)
	t.stop = t.after(wait, func() { t.fire(gen) })
}

func (t *Timer) state() State {
	st := State{Phase: t.phase, Paused: t.paused, Settings: t.settings, Stage: t.stage, NextMs: t.next.Milliseconds()}
	if t.day == t.today() {
		st.Count = t.count
	}
	if t.breakDue {
		st.Pending = t.breakPhase()
	}
	if t.phase != Idle {
		left := t.remaining
		if !t.paused {
			left = max(t.endsAt.Sub(t.wall()), 0)
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
	t.phase, t.paused, t.total, t.breakDue = p, false, total, false
	t.endsAt = t.wall().Add(d)
	t.arm()
}

// fresh starts phase p from its beginning: normal long, or the length chosen while idle, once.
func (t *Timer) fresh(p Phase, normal time.Duration) {
	d := normal
	if t.next > 0 {
		d, t.next = t.next, 0
	}
	t.run(p, d, d)
}

func (t *Timer) idle() {
	t.cancel()
	t.phase, t.paused, t.remaining, t.total, t.breakDue = Idle, false, 0, 0, false
}

func minutes(m int) time.Duration { return time.Duration(m) * time.Minute }

// breakPhase is the break that follows the focuses done: the long one closes the cycle.
func (t *Timer) breakPhase() Phase {
	if t.stage >= t.settings.Every {
		return Long
	}
	return Break
}

// runBreak starts the break that follows the focuses done; the long one starts the cycle over.
func (t *Timer) runBreak() {
	if t.breakPhase() == Long {
		t.stage = 0
		t.fresh(Long, minutes(t.settings.Long))
		return
	}
	t.fresh(Break, minutes(t.settings.Break))
}

func (t *Timer) fire(gen int) {
	t.mu.Lock()
	if gen != t.gen || t.phase == Idle || t.paused {
		t.mu.Unlock()
		return
	}
	if t.wall().Before(t.endsAt) {
		t.arm()
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
		t.idle()
		t.breakDue = true
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
		if t.breakDue {
			t.runBreak()
		} else {
			t.fresh(Focus, minutes(t.settings.Focus))
		}
	case "pause":
		if t.phase == Idle || t.paused {
			t.mu.Unlock()
			return errors.New("nada rodando para pausar")
		}
		t.remaining = max(t.endsAt.Sub(t.wall()), 0)
		t.cancel()
		t.paused = true
	case "resume":
		if !t.paused {
			t.mu.Unlock()
			return errors.New("nada pausado")
		}
		t.run(t.phase, t.remaining, t.total)
	case "skip":
		// Idle, it drops the break that is due, and the next start is a focus.
		if t.phase == Idle && !t.breakDue {
			t.mu.Unlock()
			return errors.New("nada para pular")
		}
		t.idle()
	case "reset":
		t.idle()
		t.count, t.stage, t.next = 0, 0, 0
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
		t.fresh(Focus, minutes(t.settings.Focus))
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

// SetRemaining sets the time left: of the phase running or paused, or, while idle, of the next
// phase started.
func (t *Timer) SetRemaining(d time.Duration) error {
	if d < time.Second || d > 180*time.Minute {
		return errors.New("tempo de 1 s a 180 min")
	}
	t.mu.Lock()
	switch {
	case t.phase == Idle:
		t.next = d
	case t.paused:
		t.remaining = d
	default:
		t.run(t.phase, d, t.total)
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
