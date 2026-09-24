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
