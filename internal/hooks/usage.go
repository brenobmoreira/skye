package hooks

import (
	"encoding/json"
	"strings"
	"time"
)

// Usage is the plan usage the Claude Code status line reports; it is per account, so every
// terminal reports the same numbers.
type Usage struct {
	FiveHour  *Window   `json:"fiveHour"`
	SevenDay  *Window   `json:"sevenDay"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Window struct {
	UsedPct  float64 `json:"usedPct"`
	ResetsAt int64   `json:"resetsAt"`
}

type statusPayload struct {
	RateLimits struct {
		FiveHour *rawWindow `json:"five_hour"`
		SevenDay *rawWindow `json:"seven_day"`
	} `json:"rate_limits"`
}

type rawWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	ResetsAt       float64  `json:"resets_at"`
}

func (w *rawWindow) window() *Window {
	if w == nil || w.UsedPercentage == nil {
		return nil
	}
	return &Window{UsedPct: *w.UsedPercentage, ResetsAt: int64(w.ResetsAt)}
}

func ParseUsage(body []byte) (Usage, bool) {
	var p statusPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return Usage{}, false
	}
	u := Usage{FiveHour: p.RateLimits.FiveHour.window(), SevenDay: p.RateLimits.SevenDay.window()}
	return u, u.FiveHour != nil || u.SevenDay != nil
}

const statusMarker = "http://skye/statusline"

// The status line command gets the session JSON on stdin and prints the line. The wrapper sends
// a copy to skye in the background and hands the same input to the user's own command.
const (
	statusPrefix = `in=$(cat); [ -n "$SKYE_TERMINAL_ID" ] && printf '%s' "$in" | curl -s -m 1 --unix-socket "$SKYE_SOCKET" -X POST "` +
		statusMarker + `?t=$SKYE_TERMINAL_ID" -H 'Content-Type: application/json' -d @- >/dev/null 2>&1 & printf '%s' "$in" | {` + "\n"
	statusSuffix = "\n}"
)

func StatusLineCommand(original string) string {
	if strings.TrimSpace(original) == "" {
		original = "true"
	}
	return statusPrefix + original + statusSuffix
}

func originalStatusLine(cmd string) (string, bool) {
	if !strings.HasPrefix(cmd, statusPrefix) || !strings.HasSuffix(cmd, statusSuffix) {
		return "", false
	}
	return strings.TrimSuffix(strings.TrimPrefix(cmd, statusPrefix), statusSuffix), true
}

// wrapStatusLine returns the statusLine setting with skye's relay around the user's command.
// A relay from an older skye it cannot unwrap is left alone.
func wrapStatusLine(v any) any {
	sl, ok := v.(map[string]any)
	if v == nil || !ok {
		if v != nil {
			return v
		}
		return map[string]any{"type": "command", "command": StatusLineCommand("")}
	}
	if t, _ := sl["type"].(string); t != "" && t != "command" {
		return sl
	}
	cmd, _ := sl["command"].(string)
	if strings.Contains(cmd, statusMarker) {
		original, ok := originalStatusLine(cmd)
		if !ok {
			return sl
		}
		cmd = original
	}
	sl["type"] = "command"
	sl["command"] = StatusLineCommand(cmd)
	return sl
}
