package tmuxctl

import (
	"bytes"
	"testing"
)

func TestDecodeOctal(t *testing.T) {
	cases := map[string][]byte{
		"plain":             []byte("plain"),
		`\033[1mhi\015\012`: []byte("\x1b[1mhi\r\n"),
		`back\134slash`:     []byte(`back\slash`),
		"você":              []byte("você"),
		`trailing\03`:       []byte(`trailing\03`),
		`not\89x`:           []byte(`not\89x`),
		"":                  {},
	}
	for in, want := range cases {
		if got := DecodeOctal(in); !bytes.Equal(got, want) {
			t.Errorf("DecodeOctal(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseOutput(t *testing.T) {
	l := ParseLine(`%output %3 hello\015\012`)
	if l.Kind != KindOutput || l.Pane != "%3" || string(l.Data) != "hello\r\n" {
		t.Fatalf("got %+v", l)
	}
	empty := ParseLine("%output %3 ")
	if empty.Kind != KindOutput || len(empty.Data) != 0 {
		t.Fatalf("got %+v", empty)
	}
}

func TestParseBlockMarkers(t *testing.T) {
	cases := []struct {
		in    string
		kind  Kind
		num   string
		flags string
	}{
		{"%begin 1790187763 274 1", KindBegin, "274", "1"},
		{"%end 1790187763 274 1", KindEnd, "274", "1"},
		{"%error 1790187763 278 1", KindError, "278", "1"},
		{"%begin 1790187763 269 0", KindBegin, "269", "0"},
	}
	for _, c := range cases {
		l := ParseLine(c.in)
		if l.Kind != c.kind || l.Number != c.num || l.Flags != c.flags {
			t.Errorf("ParseLine(%q) = %+v", c.in, l)
		}
	}
}

func TestParseWindowCloseAndExit(t *testing.T) {
	for _, in := range []string{"%window-close @4", "%unlinked-window-close @4"} {
		l := ParseLine(in)
		if l.Kind != KindWindowClose || l.Window != "@4" {
			t.Errorf("ParseLine(%q) = %+v", in, l)
		}
	}
	// Edge case: window-close without window ID should not panic
	for _, in := range []string{"%window-close ", "%unlinked-window-close "} {
		l := ParseLine(in)
		if l.Kind != KindWindowClose || l.Window != "" {
			t.Errorf("ParseLine(%q) = %+v, want Kind=KindWindowClose, Window=\"\"", in, l)
		}
	}
	for _, in := range []string{"%exit", "%exit server exited"} {
		if ParseLine(in).Kind != KindExit {
			t.Errorf("%q is not exit", in)
		}
	}
	for _, in := range []string{"%session-changed $0 skye", "%layout-change @1 x", "random"} {
		if ParseLine(in).Kind != KindOther {
			t.Errorf("%q should be other", in)
		}
	}
}
