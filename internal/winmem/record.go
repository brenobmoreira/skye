package winmem

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Record appends one reading to dir/<day>.jsonl, in the fields mcp-sysagent uses for the WSL, so
// procs.History reads both the same way.
func Record(dir string, m Memory, at time.Time) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	line, err := json.Marshal(struct {
		TS         time.Time `json:"ts"`
		MemTotalMB int       `json:"mem_total_mb"`
		MemAvailMB int       `json:"mem_avail_mb"`
	}{at, m.TotalMB, m.AvailMB})
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, at.Format(time.DateOnly)+".jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(append(line, '\n')); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// Prune removes the day files older than keep days; files not named after a day stay.
func Prune(dir string, now time.Time, keep int) {
	cutoff := now.AddDate(0, 0, -keep).Format(time.DateOnly)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		day, ok := strings.CutSuffix(e.Name(), ".jsonl")
		if !ok {
			continue
		}
		if _, err := time.Parse(time.DateOnly, day); err != nil {
			continue
		}
		if day <= cutoff {
			os.Remove(filepath.Join(dir, e.Name()))
		}
	}
}
