package procs

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

func write(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

type fakeProc struct {
	pid, ppid, uid int
	name, args     string
	rssKB          int
	env            []string
	cwd            string
}

func fakeRoot(t *testing.T, list ...fakeProc) string {
	t.Helper()
	root := t.TempDir()
	write(t, filepath.Join(root, "meminfo"), "MemTotal:       11216896 kB\nMemFree:          100 kB\nMemAvailable:    8969216 kB\nSwapTotal:      16777216 kB\nSwapFree:       16769024 kB\n")
	write(t, filepath.Join(root, "pressure", "memory"), "some avg10=0.00 avg60=1.25 avg300=0.10 total=6720203\nfull avg10=0.00 avg60=0.50 avg300=0.00 total=6629071\n")
	write(t, filepath.Join(root, "loadavg"), "0.52 0.40 0.30 2/900 12345\n")
	write(t, filepath.Join(root, "self", "status"), "Name:\tfake\n")
	for _, p := range list {
		dir := filepath.Join(root, itoa(p.pid))
		status := "Name:\t" + p.name + "\nPPid:\t" + itoa(p.ppid) + "\nUid:\t" + itoa(p.uid) + "\t" + itoa(p.uid) + "\t" + itoa(p.uid) + "\t" + itoa(p.uid) + "\n"
		if p.rssKB > 0 {
			status += "VmRSS:\t  " + itoa(p.rssKB) + " kB\n"
		}
		write(t, filepath.Join(dir, "status"), status)
		write(t, filepath.Join(dir, "cmdline"), strings.ReplaceAll(p.args, " ", "\x00")+"\x00")
		write(t, filepath.Join(dir, "environ"), strings.Join(p.env, "\x00")+"\x00")
		if p.cwd != "" {
			if err := os.Symlink(p.cwd, filepath.Join(dir, "cwd")); err != nil {
				t.Fatal(err)
			}
		}
	}
	return root
}

var itoa = strconv.Itoa

func TestMachineReadsMemoryPressureAndLoad(t *testing.T) {
	m, err := Reader{Root: fakeRoot(t)}.Machine()
	if err != nil {
		t.Fatal(err)
	}
	want := Machine{MemTotalMB: 10954, MemAvailMB: 8759, SwapTotalMB: 16384, SwapUsedMB: 8, PSISome60: 1.25, Load1: 0.52}
	if m != want {
		t.Fatalf("machine = %+v", m)
	}
}

func TestListKeepsTheUsersProcessesWithTheirDetails(t *testing.T) {
	root := fakeRoot(t,
		fakeProc{pid: 10, ppid: 1, uid: 1000, name: "bash", args: "/bin/bash -li", rssKB: 9344, env: []string{"HOME=/h", "SKYE_TERMINAL_ID=t1"}, cwd: "/h/projects/skye"},
		fakeProc{pid: 11, ppid: 10, uid: 1000, name: "claude", args: "claude --resume", rssKB: 337076, env: []string{"SKYE_TERMINAL_ID=t1"}},
		fakeProc{pid: 12, ppid: 1, uid: 0, name: "root-thing", args: "x", rssKB: 5},
		fakeProc{pid: 13, ppid: 2, uid: 1000, name: "kworker", args: ""},
	)
	list, err := Reader{Root: root, UID: 1000}.List()
	if err != nil {
		t.Fatal(err)
	}
	sort.Slice(list, func(i, j int) bool { return list[i].PID < list[j].PID })
	want := []Proc{
		{PID: 10, PPID: 1, Name: "bash", Args: "/bin/bash -li", RSSKB: 9344, Cwd: "/h/projects/skye", Terminal: "t1"},
		{PID: 11, PPID: 10, Name: "claude", Args: "claude --resume", RSSKB: 337076, Terminal: "t1"},
		{PID: 13, PPID: 2, Name: "kworker"},
	}
	if !reflect.DeepEqual(list, want) {
		t.Fatalf("list = %+v", list)
	}
}

func TestTreeFollowsChildren(t *testing.T) {
	table := Index([]Proc{{PID: 1}, {PID: 10, PPID: 1, RSSKB: 100}, {PID: 11, PPID: 10, RSSKB: 200}, {PID: 12, PPID: 11, RSSKB: 300}, {PID: 20, PPID: 1, RSSKB: 5}})
	got := table.Tree(10)
	sort.Ints(got)
	if !reflect.DeepEqual(got, []int{10, 11, 12}) {
		t.Fatalf("tree = %v", got)
	}
	if kb := table.RSSKB(got); kb != 600 {
		t.Fatalf("rss = %d", kb)
	}
	if got := table.Tree(99); got != nil {
		t.Fatalf("tree of missing pid = %v", got)
	}
}

func TestHistoryReadsTheSysagentSamplesInsideTheWindow(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 9, 24, 1, 0, 0, 0, time.Local)
	write(t, filepath.Join(dir, "2026-09-23.jsonl"),
		`{"schema":1,"ts":"2026-09-23T21:00:00`+zone(now)+`","mem_avail_mb":1}`+"\n"+
			`{"schema":1,"ts":"2026-09-23T23:30:00`+zone(now)+`","mem_avail_mb":7000}`+"\n")
	write(t, filepath.Join(dir, "2026-09-24.jsonl"),
		`{"schema":1,"ts":"2026-09-24T00:30:00`+zone(now)+`","mem_avail_mb":6500}`+"\n"+"not json\n")
	got := History(dir, now, 3*time.Hour)
	if len(got) != 2 || got[0].AvailMB != 7000 || got[1].AvailMB != 6500 {
		t.Fatalf("history = %+v", got)
	}
	if History(filepath.Join(dir, "missing"), now, time.Hour) != nil {
		t.Fatal("history without files")
	}
}

func zone(t time.Time) string { return t.Format("-07:00") }
