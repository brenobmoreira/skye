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

// Session is what the status line says about one claude session: how full its context is, the
// model and what it cost so far.
type Session struct {
	ContextPct *float64 `json:"contextPct,omitempty"`
	Model      string   `json:"model,omitempty"`
	CostUSD    *float64 `json:"costUsd,omitempty"`
}

// Status is one status line report: the account usage, when it came, and the session.
type Status struct {
	Provider   string
	Usage      Usage
	HasUsage   bool
	Session    Session
	AgentState string
	SessionID  string
	Cwd        string
	Title      string
	Ask        string
}

type statusPayload struct {
	ConversationID    string `json:"conversation_id"`
	SessionID         string `json:"session_id"`
	Cwd               string `json:"cwd"`
	AgentState        string `json:"agent_state"`
	Ask               string `json:"ask"`
	InitialPrompt     string `json:"initial_prompt"`
	ConversationTitle string `json:"conversation_title"`
	Model             struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage      *float64 `json:"used_percentage"`
		RemainingPercentage *float64 `json:"remaining_percentage"`
	} `json:"context_window"`
	Cost struct {
		TotalCostUSD *float64 `json:"total_cost_usd"`
		TotalUSD     *float64 `json:"total_usd"`
	} `json:"cost"`
	RateLimits struct {
		FiveHour  *rawWindow `json:"five_hour"`
		SevenDay  *rawWindow `json:"seven_day"`
		Primary   *rawWindow `json:"primary"`
		Secondary *rawWindow `json:"secondary"`
	} `json:"rate_limits"`
	Quota map[string]rawQuotaBucket `json:"quota"`
}

type rawWindow struct {
	UsedPercentage *float64 `json:"used_percentage"`
	UsedPercent    *float64 `json:"used_percent"`
	ResetsAt       float64  `json:"resets_at"`
}

func (w *rawWindow) window() *Window {
	if w == nil {
		return nil
	}
	pct := w.UsedPercentage
	if pct == nil {
		pct = w.UsedPercent
	}
	if pct == nil {
		return nil
	}
	return &Window{UsedPct: *pct, ResetsAt: int64(w.ResetsAt)}
}

type rawQuotaBucket struct {
	UsedPercentage     *float64 `json:"used_percentage"`
	RemainingFraction  *float64 `json:"remaining_fraction"`
	ResetsAt           float64  `json:"resets_at"`
	ResetTime          string   `json:"reset_time"`
	ResetInSeconds     *float64 `json:"reset_in_seconds"`
	Window             string   `json:"window"`
	RefreshWindowLabel string   `json:"refresh_window_label"`
	Disabled           bool     `json:"disabled"`
}

func (b rawQuotaBucket) window(now time.Time) (*Window, bool) {
	if b.Disabled {
		return nil, false
	}
	var used float64
	switch {
	case b.UsedPercentage != nil:
		used = *b.UsedPercentage
	case b.RemainingFraction != nil:
		used = (1.0 - *b.RemainingFraction) * 100
	default:
		return nil, false
	}
	var resets int64
	switch {
	case b.ResetsAt > 0:
		resets = int64(b.ResetsAt)
	case b.ResetTime != "":
		if t, err := time.Parse(time.RFC3339, b.ResetTime); err == nil {
			resets = t.Unix()
		}
	case b.ResetInSeconds != nil && *b.ResetInSeconds > 0:
		resets = now.Unix() + int64(*b.ResetInSeconds)
	}
	return &Window{UsedPct: used, ResetsAt: resets}, true
}

func isWeeklyBucket(key string, b rawQuotaBucket) bool {
	label := strings.ToLower(key + " " + b.Window + " " + b.RefreshWindowLabel)
	if strings.Contains(label, "7d") || strings.Contains(label, "seven") || strings.Contains(label, "week") {
		return true
	}
	return b.ResetInSeconds != nil && *b.ResetInSeconds > 86400
}

func ParseUsage(body []byte) (Usage, bool) {
	s, _ := ParseStatus(body)
	return s.Usage, s.HasUsage
}

func ParseStatus(body []byte) (Status, bool) {
	var p statusPayload
	if err := json.Unmarshal(body, &p); err != nil {
		return Status{}, false
	}
	fh := p.RateLimits.FiveHour.window()
	if fh == nil {
		fh = p.RateLimits.Primary.window()
	}
	sd := p.RateLimits.SevenDay.window()
	if sd == nil {
		sd = p.RateLimits.Secondary.window()
	}
	now := time.Now()
	for k, b := range p.Quota {
		w, ok := b.window(now)
		if !ok {
			continue
		}
		if isWeeklyBucket(k, b) {
			if sd == nil || w.UsedPct > sd.UsedPct {
				sd = w
			}
		} else {
			if fh == nil || w.UsedPct > fh.UsedPct {
				fh = w
			}
		}
	}
	u := Usage{FiveHour: fh, SevenDay: sd}
	ctxPct := p.ContextWindow.UsedPercentage
	if ctxPct == nil && p.ContextWindow.RemainingPercentage != nil {
		v := 100.0 - *p.ContextWindow.RemainingPercentage
		ctxPct = &v
	}
	cost := p.Cost.TotalCostUSD
	if cost == nil {
		cost = p.Cost.TotalUSD
	}
	var provider string
	switch {
	case len(p.Quota) > 0 || p.AgentState != "" || p.ConversationID != "":
		provider = "agy"
	case p.RateLimits.Primary != nil || p.RateLimits.Secondary != nil:
		provider = "codex"
	case p.RateLimits.FiveHour != nil || p.RateLimits.SevenDay != nil || p.Cost.TotalCostUSD != nil:
		provider = "claude"
	}
	sessionID := p.ConversationID
	if sessionID == "" {
		sessionID = p.SessionID
	}
	title := p.ConversationTitle
	if title == "" {
		title = p.InitialPrompt
	}
	return Status{
		Provider:   provider,
		Usage:      u,
		HasUsage:   u.FiveHour != nil || u.SevenDay != nil,
		Session:    Session{ContextPct: ctxPct, Model: p.Model.DisplayName, CostUSD: cost},
		AgentState: p.AgentState,
		SessionID:  sessionID,
		Cwd:        p.Cwd,
		Title:      title,
		Ask:        p.Ask,
	}, true
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
