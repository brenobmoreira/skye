package pomodoro

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

type scheduled struct {
	d       time.Duration
	f       func()
	stopped bool
}

type rig struct {
	now     time.Time
	jobs    []*scheduled
	changes []State
	ends    []Ended
	t       *Timer
}

func newRig(t *testing.T, s Settings) *rig {
	t.Helper()
	r := &rig{now: time.Date(2026, 11, 11, 10, 0, 0, 0, time.Local)}
	after := func(d time.Duration, f func()) func() bool {
		j := &scheduled{d: d, f: f}
		r.jobs = append(r.jobs, j)
		return func() bool { was := !j.stopped; j.stopped = true; return was }
	}
	r.t = New(s, func() time.Time { return r.now }, after, func(st State) { r.changes = append(r.changes, st) }, func(e Ended) { r.ends = append(r.ends, e) })
	return r
}

// fire moves the clock to the end of the last scheduled job and runs it.
func (r *rig) fire(t *testing.T) {
	t.Helper()
	j := r.jobs[len(r.jobs)-1]
	if j.stopped {
		t.Fatal("firing a stopped job")
	}
	r.now = r.now.Add(j.d)
	j.f()
}

func (r *rig) do(t *testing.T, action string) {
	t.Helper()
	if err := r.t.Do(action); err != nil {
		t.Fatalf("%s: %v", action, err)
	}
}

func TestStartRunsAFocus(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	st := r.t.State()
	if st.Phase != "focus" || st.RemainingMs != 25*60_000 || st.TotalMs != 25*60_000 || st.Paused {
		t.Fatalf("state %+v", st)
	}
	if len(r.changes) != 1 {
		t.Fatalf("changes = %d", len(r.changes))
	}
	if err := r.t.Do("start"); err == nil {
		t.Fatal("start while running accepted")
	}
	if err := r.t.Do("jump"); err == nil {
		t.Fatal("unknown action accepted")
	}
}

func TestFocusEndStartsABreak(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	r.fire(t)
	st := r.t.State()
	if st.Count != 1 || st.Phase != "break" || st.RemainingMs != 5*60_000 {
		t.Fatalf("state %+v", st)
	}
	if len(r.ends) != 1 || r.ends[0].Phase != "focus" || r.ends[0].Next.Phase != "break" {
		t.Fatalf("ends %+v", r.ends)
	}
}

func TestEveryNthFocusGetsTheLongBreak(t *testing.T) {
	r := newRig(t, Settings{Focus: 25, Break: 5, Long: 15, Every: 2})
	r.do(t, "start")
	r.fire(t) // focus 1 → break
	r.fire(t) // break → idle
	r.do(t, "start")
	r.fire(t) // focus 2 → long
	st := r.t.State()
	if st.Phase != "long" || st.RemainingMs != 15*60_000 || st.Count != 2 {
		t.Fatalf("state %+v", st)
	}
}

func TestBreakEndWaitsForStart(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	r.fire(t)
	jobs := len(r.jobs)
	r.fire(t)
	st := r.t.State()
	if st.Phase != "idle" || st.RemainingMs != 0 {
		t.Fatalf("state %+v", st)
	}
	if r.ends[len(r.ends)-1].Phase != "break" {
		t.Fatalf("ends %+v", r.ends)
	}
	if len(r.jobs) != jobs {
		t.Fatal("something was scheduled after the break")
	}
}

func TestPauseAndResume(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	r.now = r.now.Add(10 * time.Minute)
	r.do(t, "pause")
	st := r.t.State()
	if !st.Paused || st.RemainingMs != 15*60_000 {
		t.Fatalf("state %+v", st)
	}
	if !r.jobs[0].stopped {
		t.Fatal("job not stopped on pause")
	}
	r.now = r.now.Add(time.Hour)
	if st := r.t.State(); st.RemainingMs != 15*60_000 {
		t.Fatalf("time ran while paused: %+v", st)
	}
	r.do(t, "resume")
	if r.jobs[len(r.jobs)-1].d != 15*time.Minute {
		t.Fatalf("rescheduled for %v", r.jobs[len(r.jobs)-1].d)
	}
	if st := r.t.State(); st.Paused || st.TotalMs != 25*60_000 {
		t.Fatalf("state %+v", st)
	}
	if err := r.t.Do("resume"); err == nil {
		t.Fatal("resume while running accepted")
	}
}

func TestSkipDoesNotCount(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	r.do(t, "skip")
	st := r.t.State()
	if st.Phase != "idle" || st.Count != 0 || len(r.ends) != 0 || !r.jobs[0].stopped {
		t.Fatalf("state %+v ends %v", st, r.ends)
	}
	r.do(t, "start")
	r.fire(t)
	r.do(t, "skip")
	if st := r.t.State(); st.Phase != "idle" || st.Count != 1 || len(r.ends) != 1 {
		t.Fatalf("state %+v ends %v", st, r.ends)
	}
}

func TestResetZeroesTheDay(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	r.fire(t)
	r.do(t, "reset")
	if st := r.t.State(); st.Phase != "idle" || st.Count != 0 {
		t.Fatalf("state %+v", st)
	}
}

func TestFocusEndingAfterMidnightCountsForTheNewDay(t *testing.T) {
	r := newRig(t, Settings{Focus: 25, Break: 5, Long: 15, Every: 12})
	for range 3 {
		r.do(t, "start")
		r.fire(t)
		r.do(t, "skip")
	}
	if c := r.t.State().Count; c != 3 {
		t.Fatalf("count = %d", c)
	}
	r.now = time.Date(2026, 11, 11, 23, 50, 0, 0, time.Local)
	r.do(t, "start")
	r.fire(t)
	if st := r.t.State(); st.Count != 1 {
		t.Fatalf("count after midnight = %d", st.Count)
	}

	q := newRig(t, Default)
	q.do(t, "start")
	q.fire(t)
	q.do(t, "skip")
	q.now = q.now.AddDate(0, 0, 1)
	if c := q.t.State().Count; c != 0 {
		t.Fatalf("yesterday's count leaked: %d", c)
	}
}

func TestSettingsChangeAppliesToTheNextPhase(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	if err := r.t.SetSettings(Settings{Focus: 50, Break: 5, Long: 15, Every: 4}); err != nil {
		t.Fatal(err)
	}
	if st := r.t.State(); st.RemainingMs != 25*60_000 || st.Settings.Focus != 50 {
		t.Fatalf("state %+v", st)
	}
	r.do(t, "skip")
	r.do(t, "start")
	if st := r.t.State(); st.RemainingMs != 50*60_000 {
		t.Fatalf("state %+v", st)
	}
	if err := r.t.SetSettings(Settings{Focus: 0, Break: 5, Long: 15, Every: 4}); err == nil {
		t.Fatal("zero minutes accepted")
	}
}

func TestSettingsValidate(t *testing.T) {
	bad := []Settings{{0, 5, 15, 4}, {181, 5, 15, 4}, {25, 0, 15, 4}, {25, 5, 0, 4}, {25, 5, 15, 0}, {25, 5, 15, 13}}
	for _, s := range bad {
		if s.Validate() == nil {
			t.Errorf("%+v accepted", s)
		}
	}
	if err := Default.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestLoadSettings(t *testing.T) {
	file := filepath.Join(t.TempDir(), "pomodoro.json")
	if s, err := LoadSettings(file); err != nil || s != Default {
		t.Fatalf("missing: %+v %v", s, err)
	}
	os.WriteFile(file, []byte("{"), 0o600)
	if s, err := LoadSettings(file); err == nil || s != Default {
		t.Fatalf("broken: %+v %v", s, err)
	}
	want := Settings{Focus: 50, Break: 10, Long: 20, Every: 3}
	if err := SaveSettings(file, want); err != nil {
		t.Fatal(err)
	}
	if s, err := LoadSettings(file); err != nil || s != want {
		t.Fatalf("round trip: %+v %v", s, err)
	}
}

func TestBeginFocusAtAStage(t *testing.T) {
	r := newRig(t, Default)
	if err := r.t.Begin(Focus, 3); err != nil {
		t.Fatal(err)
	}
	st := r.t.State()
	if st.Phase != Focus || st.Stage != 2 || st.Count != 0 || st.RemainingMs != 25*60_000 {
		t.Fatalf("state %+v", st)
	}
	r.fire(t)
	if st := r.t.State(); st.Phase != Break || st.Stage != 3 || st.Count != 1 {
		t.Fatalf("after focus %+v", st)
	}
}

func TestBeginFocusAtTheLastStageEndsInTheLongBreak(t *testing.T) {
	r := newRig(t, Default)
	r.t.Begin(Focus, 4)
	r.fire(t)
	if st := r.t.State(); st.Phase != Long || st.Stage != 0 || st.RemainingMs != 15*60_000 {
		t.Fatalf("state %+v", st)
	}
}

func TestBeginBreakAfterAStage(t *testing.T) {
	r := newRig(t, Default)
	if err := r.t.Begin(Break, 2); err != nil {
		t.Fatal(err)
	}
	if st := r.t.State(); st.Phase != Break || st.Stage != 2 || st.RemainingMs != 5*60_000 || st.Count != 0 {
		t.Fatalf("state %+v", st)
	}
	r.do(t, "skip")
	if err := r.t.Begin(Break, 4); err != nil {
		t.Fatal(err)
	}
	if st := r.t.State(); st.Phase != Long || st.Stage != 0 || st.RemainingMs != 15*60_000 {
		t.Fatalf("state %+v", st)
	}
	r.fire(t)
	r.do(t, "start")
	if st := r.t.State(); st.Stage != 0 {
		t.Fatalf("after the long break the cycle starts over: %+v", st)
	}
}

func TestBeginValidates(t *testing.T) {
	r := newRig(t, Default)
	for _, c := range []struct {
		p Phase
		s int
	}{{Focus, 0}, {Focus, 5}, {Break, 0}, {Idle, 1}, {Long, 4}} {
		if err := r.t.Begin(c.p, c.s); err == nil {
			t.Errorf("Begin(%s, %d) accepted", c.p, c.s)
		}
	}
	r.t.Begin(Focus, 1)
	if err := r.t.Begin(Focus, 2); err == nil {
		t.Fatal("Begin while running accepted")
	}
}

func TestStageFollowsTheCycle(t *testing.T) {
	r := newRig(t, Settings{Focus: 25, Break: 5, Long: 15, Every: 2})
	r.do(t, "start")
	r.fire(t)
	if st := r.t.State(); st.Stage != 1 || st.Phase != Break {
		t.Fatalf("state %+v", st)
	}
	r.fire(t)
	r.do(t, "start")
	r.fire(t)
	if st := r.t.State(); st.Stage != 0 || st.Phase != Long {
		t.Fatalf("state %+v", st)
	}
	r.do(t, "reset")
	if st := r.t.State(); st.Stage != 0 {
		t.Fatalf("reset kept the stage: %+v", st)
	}
}

func TestSetRemainingWhileRunningOrPaused(t *testing.T) {
	r := newRig(t, Default)
	r.do(t, "start")
	if err := r.t.SetRemaining(18 * time.Minute); err != nil {
		t.Fatal(err)
	}
	st := r.t.State()
	if st.RemainingMs != 18*60_000 || st.TotalMs != 25*60_000 || !r.jobs[0].stopped {
		t.Fatalf("state %+v", st)
	}
	if last := r.jobs[len(r.jobs)-1]; last.d != 18*time.Minute {
		t.Fatalf("rescheduled for %v", last.d)
	}
	r.do(t, "pause")
	r.t.SetRemaining(5 * time.Minute)
	if st := r.t.State(); st.RemainingMs != 5*60_000 || !st.Paused {
		t.Fatalf("paused %+v", st)
	}
	r.do(t, "resume")
	if last := r.jobs[len(r.jobs)-1]; last.d != 5*time.Minute {
		t.Fatalf("resumed for %v", last.d)
	}
}

func TestSetRemainingWhileIdleAppliesToTheNextStartOnly(t *testing.T) {
	r := newRig(t, Default)
	if err := r.t.SetRemaining(18 * time.Minute); err != nil {
		t.Fatal(err)
	}
	if st := r.t.State(); st.Phase != Idle || st.NextMs != 18*60_000 {
		t.Fatalf("idle %+v", st)
	}
	r.do(t, "start")
	if st := r.t.State(); st.RemainingMs != 18*60_000 || st.TotalMs != 18*60_000 || st.NextMs != 0 {
		t.Fatalf("started %+v", st)
	}
	r.do(t, "skip")
	r.t.Begin(Focus, 1)
	if st := r.t.State(); st.RemainingMs != 25*60_000 {
		t.Fatalf("second start %+v", st)
	}
}

func TestSetRemainingValidates(t *testing.T) {
	r := newRig(t, Default)
	for _, d := range []time.Duration{0, -time.Second, 181 * time.Minute} {
		if err := r.t.SetRemaining(d); err == nil {
			t.Errorf("%v accepted", d)
		}
	}
}

func TestResetForgetsTheChosenLength(t *testing.T) {
	r := newRig(t, Default)
	r.t.SetRemaining(18 * time.Minute)
	r.do(t, "reset")
	if st := r.t.State(); st.NextMs != 0 {
		t.Fatalf("state %+v", st)
	}
}
