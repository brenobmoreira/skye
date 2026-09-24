// Package winmem reads the Windows host's memory from inside WSL. The Linux side only sees the
// VM; the memory that runs out first is the host's, which also holds what the VM keeps as cache.
package winmem

import (
	"bufio"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Memory struct {
	TotalMB   int       `json:"totalMb"`
	AvailMB   int       `json:"availMb"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// Script prints "total free" in KB every five seconds for five minutes, then exits, so a skye
// that dies without stopping it does not leave it running; the watcher starts another while
// someone still looks. It goes to powershell.exe on stdin, with nothing encoded on its command line.
const Script = `for ($i = 0; $i -lt 60; $i++) { $o = Get-CimInstance Win32_OperatingSystem; [Console]::Out.WriteLine("$($o.TotalVisibleMemorySize) $($o.FreePhysicalMemory)"); [Console]::Out.Flush(); Start-Sleep -Seconds 5 }` + "\n"

func Parse(line string) (Memory, bool) {
	f := strings.Fields(line)
	if len(f) != 2 {
		return Memory{}, false
	}
	total, err1 := strconv.Atoi(f[0])
	free, err2 := strconv.Atoi(f[1])
	if err1 != nil || err2 != nil || total <= 0 {
		return Memory{}, false
	}
	return Memory{TotalMB: total / 1024, AvailMB: free / 1024}, true
}

// Watcher keeps one reader running while Latest is being called and stops it after Idle without
// calls.
type Watcher struct {
	Start func() (<-chan string, func(), error)
	Idle  time.Duration
	Now   func() time.Time

	mu      sync.Mutex
	running bool
	stop    func()
	gen     int
	asked   time.Time
	last    Memory
}

// New watches the host through powershell.exe, or returns nil outside WSL.
func New() *Watcher {
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		return nil
	}
	w := &Watcher{Start: startPowershell, Idle: 30 * time.Second, Now: time.Now}
	go func() {
		for range time.Tick(10 * time.Second) {
			w.Reap()
		}
	}()
	return w
}

// Latest starts the reader if needed and returns the last reading, when it is still fresh.
func (w *Watcher) Latest() (Memory, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()
	now := w.Now()
	w.asked = now
	if !w.running {
		if lines, stop, err := w.Start(); err == nil {
			w.running, w.stop = true, stop
			w.gen++
			go w.read(lines, w.gen)
		}
	}
	if w.last.TotalMB == 0 || now.Sub(w.last.UpdatedAt) > w.Idle {
		return Memory{}, false
	}
	return w.last, true
}

func (w *Watcher) read(lines <-chan string, gen int) {
	for line := range lines {
		m, ok := Parse(line)
		if !ok {
			continue
		}
		w.mu.Lock()
		if gen == w.gen {
			m.UpdatedAt = w.Now()
			w.last = m
		}
		w.mu.Unlock()
	}
	w.mu.Lock()
	if gen == w.gen {
		w.running = false
	}
	w.mu.Unlock()
}

// Reap stops the reader once nobody asked for Idle.
func (w *Watcher) Reap() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running && w.Now().Sub(w.asked) > w.Idle {
		w.halt()
	}
}

func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		w.halt()
	}
}

func (w *Watcher) halt() {
	w.stop()
	w.running = false
	w.gen++
}

func startPowershell() (<-chan string, func(), error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", "-")
	cmd.Stdin = strings.NewReader(Script)
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	lines := make(chan string)
	go func() {
		defer close(lines)
		sc := bufio.NewScanner(out)
		for sc.Scan() {
			lines <- sc.Text()
		}
		_ = cmd.Wait()
	}()
	return lines, func() { _ = cmd.Process.Kill() }, nil
}
