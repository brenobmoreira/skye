// Package procs reads what the monitor shows from /proc: the machine's memory and the user's
// processes.
package procs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Machine struct {
	MemTotalMB  int     `json:"memTotalMb"`
	MemAvailMB  int     `json:"memAvailMb"`
	SwapTotalMB int     `json:"swapTotalMb"`
	SwapUsedMB  int     `json:"swapUsedMb"`
	PSISome60   float64 `json:"psiSome60"`
	Load1       float64 `json:"load1"`
}

type Proc struct {
	PID   int
	PPID  int
	Name  string
	Args  string
	RSSKB int
	Cwd   string
	// Terminal is the SKYE_TERMINAL_ID the process inherited, if any.
	Terminal string
}

// Reader reads one /proc; Root is "/proc" outside tests and UID the user whose processes count.
type Reader struct {
	Root string
	UID  int
}

func (r Reader) Machine() (Machine, error) {
	info, err := os.ReadFile(filepath.Join(r.Root, "meminfo"))
	if err != nil {
		return Machine{}, err
	}
	kb := map[string]int{}
	for _, line := range strings.Split(string(info), "\n") {
		key, rest, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		if f := strings.Fields(rest); len(f) > 0 {
			kb[key], _ = strconv.Atoi(f[0])
		}
	}
	m := Machine{
		MemTotalMB:  kb["MemTotal"] / 1024,
		MemAvailMB:  kb["MemAvailable"] / 1024,
		SwapTotalMB: kb["SwapTotal"] / 1024,
		SwapUsedMB:  (kb["SwapTotal"] - kb["SwapFree"]) / 1024,
	}
	if pressure, err := os.ReadFile(filepath.Join(r.Root, "pressure", "memory")); err == nil {
		for _, field := range strings.Fields(strings.SplitN(string(pressure), "\n", 2)[0]) {
			if v, ok := strings.CutPrefix(field, "avg60="); ok {
				m.PSISome60, _ = strconv.ParseFloat(v, 64)
			}
		}
	}
	if load, err := os.ReadFile(filepath.Join(r.Root, "loadavg")); err == nil {
		if f := strings.Fields(string(load)); len(f) > 0 {
			m.Load1, _ = strconv.ParseFloat(f[0], 64)
		}
	}
	return m, nil
}

// List returns the user's processes. A process that exits while being read is skipped.
func (r Reader) List() ([]Proc, error) {
	entries, err := os.ReadDir(r.Root)
	if err != nil {
		return nil, err
	}
	var list []Proc
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		if p, ok := r.read(pid); ok {
			list = append(list, p)
		}
	}
	return list, nil
}

func (r Reader) read(pid int) (Proc, bool) {
	dir := filepath.Join(r.Root, strconv.Itoa(pid))
	status, err := os.ReadFile(filepath.Join(dir, "status"))
	if err != nil {
		return Proc{}, false
	}
	p := Proc{PID: pid}
	uid := -1
	sc := bufio.NewScanner(bytes.NewReader(status))
	for sc.Scan() {
		key, rest, ok := strings.Cut(sc.Text(), ":")
		if !ok {
			continue
		}
		f := strings.Fields(rest)
		if len(f) == 0 {
			continue
		}
		switch key {
		case "Name":
			p.Name = f[0]
		case "PPid":
			p.PPID, _ = strconv.Atoi(f[0])
		case "Uid":
			uid, _ = strconv.Atoi(f[0])
		case "VmRSS":
			p.RSSKB, _ = strconv.Atoi(f[0])
		}
	}
	if uid != r.UID {
		return Proc{}, false
	}
	if cmdline, err := os.ReadFile(filepath.Join(dir, "cmdline")); err == nil {
		p.Args = strings.TrimSpace(strings.ReplaceAll(string(cmdline), "\x00", " "))
	}
	if environ, err := os.ReadFile(filepath.Join(dir, "environ")); err == nil {
		for _, kv := range strings.Split(string(environ), "\x00") {
			if v, ok := strings.CutPrefix(kv, "SKYE_TERMINAL_ID="); ok {
				p.Terminal = v
			}
		}
	}
	p.Cwd, _ = os.Readlink(filepath.Join(dir, "cwd"))
	return p, true
}

// Table indexes processes by pid.
type Table map[int]Proc

func Index(list []Proc) Table {
	t := make(Table, len(list))
	for _, p := range list {
		t[p.PID] = p
	}
	return t
}

// Tree returns pid and all its descendants, or nil when pid is not in the table.
func (t Table) Tree(pid int) []int {
	if _, ok := t[pid]; !ok {
		return nil
	}
	children := map[int][]int{}
	for _, p := range t {
		children[p.PPID] = append(children[p.PPID], p.PID)
	}
	out := []int{pid}
	for i := 0; i < len(out); i++ {
		out = append(out, children[out[i]]...)
	}
	return out
}

func (t Table) RSSKB(pids []int) int {
	sum := 0
	for _, pid := range pids {
		sum += t[pid].RSSKB
	}
	return sum
}

// Point is one minute of the mcp-sysagent memory history.
type Point struct {
	T       time.Time `json:"t"`
	AvailMB int       `json:"availMb"`
}

// History reads the samples mcp-sysagent keeps in dir (one YYYY-MM-DD.jsonl per day) from the
// last window (up to a day) before now. No files means no history.
func History(dir string, now time.Time, window time.Duration) []Point {
	from := now.Add(-window)
	days := []string{from.Format(time.DateOnly)}
	if today := now.Format(time.DateOnly); today != days[0] {
		days = append(days, today)
	}
	var out []Point
	for _, day := range days {
		f, err := os.Open(filepath.Join(dir, day+".jsonl"))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(f)
		sc.Buffer(make([]byte, 64*1024), 1024*1024)
		for sc.Scan() {
			var s struct {
				TS         time.Time `json:"ts"`
				MemAvailMB int       `json:"mem_avail_mb"`
			}
			if json.Unmarshal(sc.Bytes(), &s) != nil || s.TS.Before(from) || s.TS.After(now) {
				continue
			}
			out = append(out, Point{T: s.TS, AvailMB: s.MemAvailMB})
		}
		f.Close()
	}
	return out
}
