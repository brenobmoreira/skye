package winmem

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/brenobmoreira/skye/internal/procs"
)

func TestRecordWritesWhatHistoryReads(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "winmem")
	at := time.Date(2026, 11, 11, 14, 0, 0, 0, time.Local)
	for i, avail := range []int{6000, 5500, 5200} {
		if err := Record(dir, Memory{TotalMB: 16000, AvailMB: avail}, at.Add(time.Duration(i)*time.Minute)); err != nil {
			t.Fatal(err)
		}
	}
	got := procs.History(dir, at.Add(5*time.Minute), 3*time.Hour)
	if len(got) != 3 || got[0].AvailMB != 6000 || got[2].AvailMB != 5200 {
		t.Fatalf("history %+v", got)
	}
}

func TestPruneDropsOldDays(t *testing.T) {
	dir := t.TempDir()
	now := time.Date(2026, 11, 11, 14, 0, 0, 0, time.Local)
	for _, day := range []string{"2026-11-01", "2026-11-04", "2026-11-05", "2026-11-11", "notes"} {
		os.WriteFile(filepath.Join(dir, day+".jsonl"), []byte("{}\n"), 0o600)
	}
	Prune(dir, now, 7)
	left, _ := os.ReadDir(dir)
	var names []string
	for _, e := range left {
		names = append(names, e.Name())
	}
	want := "2026-11-05.jsonl 2026-11-11.jsonl notes.jsonl"
	if got := joinNames(names); got != want {
		t.Fatalf("left %s", got)
	}
}

func joinNames(n []string) string {
	out := ""
	for i, s := range n {
		if i > 0 {
			out += " "
		}
		out += s
	}
	return out
}
