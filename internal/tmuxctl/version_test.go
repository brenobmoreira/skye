package tmuxctl

import "testing"

func TestParseVersion(t *testing.T) {
	cases := map[string][2]int{
		"tmux 3.4":      {3, 4},
		"tmux 3.3a":     {3, 3},
		"tmux next-3.5": {3, 5},
		"tmux 2.9":      {2, 9},
	}
	for in, want := range cases {
		maj, min, err := ParseVersion(in)
		if err != nil || maj != want[0] || min != want[1] {
			t.Errorf("ParseVersion(%q) = %d.%d %v", in, maj, min, err)
		}
	}
	if _, _, err := ParseVersion("garbage"); err == nil {
		t.Error("expected error")
	}
}
