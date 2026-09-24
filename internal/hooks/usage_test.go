package hooks

import (
	"os/exec"
	"strings"
	"testing"
)

func TestParseUsageReadsBothWindows(t *testing.T) {
	body := []byte(`{"model":{"id":"x"},"rate_limits":{"five_hour":{"used_percentage":42.5,"resets_at":1790000000},"seven_day":{"used_percentage":18,"resets_at":1790500000}}}`)
	u, ok := ParseUsage(body)
	if !ok || u.FiveHour == nil || u.FiveHour.UsedPct != 42.5 || u.FiveHour.ResetsAt != 1790000000 || u.SevenDay == nil || u.SevenDay.UsedPct != 18 {
		t.Fatalf("usage = %+v ok=%v", u, ok)
	}
}

func TestParseUsageWithoutRateLimits(t *testing.T) {
	for _, body := range []string{`{"model":{"id":"x"}}`, `not json`, `{"rate_limits":{}}`} {
		if _, ok := ParseUsage([]byte(body)); ok {
			t.Errorf("ParseUsage(%s) ok", body)
		}
	}
}

func TestStatusLineCommandKeepsTheOriginalOutputAndCanBeUnwrapped(t *testing.T) {
	original := "echo \"$(cat | wc -c) bytes\" # mine"
	wrapped := StatusLineCommand(original)
	if got, ok := originalStatusLine(wrapped); !ok || got != original {
		t.Fatalf("unwrap = %q %v", got, ok)
	}
	out, err := exec.Command("sh", "-c", wrapped).CombinedOutput()
	if err != nil {
		t.Fatalf("%v: %s", err, out)
	}
	cmd := exec.Command("sh", "-c", wrapped)
	cmd.Stdin = strings.NewReader(`{"a":1}`)
	out, err = cmd.CombinedOutput()
	if err != nil || strings.TrimSpace(string(out)) != "7 bytes" {
		t.Fatalf("output = %q %v", out, err)
	}
}

func TestParseStatusReadsTheSession(t *testing.T) {
	body := []byte(`{"model":{"id":"claude-opus-5-5","display_name":"Opus 5.5"},"context_window":{"used_percentage":42.4},"cost":{"total_cost_usd":1.234},"rate_limits":{"five_hour":{"used_percentage":10,"resets_at":1790000000}}}`)
	s, ok := ParseStatus(body)
	if !ok || !s.HasUsage || s.Usage.FiveHour.UsedPct != 10 {
		t.Fatalf("status = %+v ok=%v", s, ok)
	}
	if s.Session.ContextPct == nil || *s.Session.ContextPct != 42.4 || s.Session.Model != "Opus 5.5" || s.Session.CostUSD == nil || *s.Session.CostUSD != 1.234 {
		t.Fatalf("session = %+v", s.Session)
	}
}

func TestParseStatusWithoutUsageOrSession(t *testing.T) {
	s, ok := ParseStatus([]byte(`{"model":{"id":"x"}}`))
	if !ok || s.HasUsage || s.Session.ContextPct != nil || s.Session.Model != "" {
		t.Fatalf("status = %+v ok=%v", s, ok)
	}
	if _, ok := ParseStatus([]byte(`not json`)); ok {
		t.Fatal("bad json parsed")
	}
}
