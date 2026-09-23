package tmuxctl

import "testing"

func TestQuote(t *testing.T) {
	q, err := Quote("/home/demo/my dir")
	if err != nil || q != "'/home/demo/my dir'" {
		t.Fatalf("got %q %v", q, err)
	}
	for _, bad := range []string{"it's", "a\nb"} {
		if _, err := Quote(bad); err == nil {
			t.Errorf("Quote(%q) should fail", bad)
		}
	}
}

func TestRenderSnapshot(t *testing.T) {
	got := RenderSnapshot([]string{"$ echo oi", "oi", "$ ", "", ""}, 2, 2)
	want := "\x1b[H\x1b[2J$ echo oi\r\noi\r\n$ \x1b[3;3H"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}
