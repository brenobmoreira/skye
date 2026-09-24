package winmem

import (
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseReadsKilobytes(t *testing.T) {
	got, ok := Parse("16429944 858404\r")
	if !ok || got.TotalMB != 16044 || got.AvailMB != 838 {
		t.Fatalf("got %+v %v", got, ok)
	}
	for _, bad := range []string{"", "x y", "12", "0 0"} {
		if _, ok := Parse(bad); ok {
			t.Errorf("Parse(%q) ok", bad)
		}
	}
}

type fakeProc struct {
	mu      sync.Mutex
	started int
	killed  int
	lines   chan string
}

func (f *fakeProc) start() (<-chan string, func(), error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.started++
	f.lines = make(chan string, 4)
	lines := f.lines
	return lines, func() { f.mu.Lock(); f.killed++; f.mu.Unlock(); close(lines) }, nil
}

func (f *fakeProc) counts() (int, int) { f.mu.Lock(); defer f.mu.Unlock(); return f.started, f.killed }

func TestWatcherStartsOnDemandKeepsTheLatestAndStopsWhenIdle(t *testing.T) {
	p := &fakeProc{}
	now := time.Unix(1000, 0)
	var mu sync.Mutex
	clock := func() time.Time { mu.Lock(); defer mu.Unlock(); return now }
	w := &Watcher{Start: p.start, Idle: 30 * time.Second, Now: clock}
	if _, ok := w.Latest(); ok {
		t.Fatal("reading before any line")
	}
	w.Latest()
	if s, _ := p.counts(); s != 1 {
		t.Fatalf("started %d times", s)
	}
	p.lines <- "16429944 858404"
	deadline := time.Now().Add(2 * time.Second)
	for {
		if got, ok := w.Latest(); ok && got.AvailMB == 838 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("line never read")
		}
		time.Sleep(5 * time.Millisecond)
	}
	mu.Lock()
	now = now.Add(31 * time.Second)
	mu.Unlock()
	w.Reap()
	if _, k := p.counts(); k != 1 {
		t.Fatalf("killed %d times", k)
	}
	w.Latest()
	if s, _ := p.counts(); s != 2 {
		t.Fatalf("not restarted: %d", s)
	}
	p.mu.Lock()
	close(p.lines)
	p.mu.Unlock()
	deadline = time.Now().Add(2 * time.Second)
	for {
		w.Latest()
		if s, _ := p.counts(); s == 3 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("not restarted after the script ended by itself")
		}
		time.Sleep(5 * time.Millisecond)
	}
	w.Stop()
}

func TestScriptEndsOnItsOwn(t *testing.T) {
	if !strings.Contains(Script, "Win32_OperatingSystem") || !strings.Contains(Script, "$i -lt 60") || strings.Contains(Script, "while ($true)") || !strings.HasSuffix(Script, "\n") || strings.Contains(Script, "\n\n") {
		t.Fatalf("script = %q", Script)
	}
}
