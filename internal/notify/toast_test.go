package notify

import (
	"encoding/base64"
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

func decode(t *testing.T, s string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		t.Fatal(err)
	}
	u := make([]uint16, len(raw)/2)
	for i := range u {
		u[i] = binary.LittleEndian.Uint16(raw[i*2:])
	}
	return string(utf16.Decode(u))
}

func TestScriptEscapesQuotes(t *testing.T) {
	s := Script("it's você", "esperando você")
	if !strings.Contains(s, "'it''s você'") || !strings.Contains(s, "'esperando você'") {
		t.Fatalf("script = %s", s)
	}
}

func TestEncodeRoundTripsUnicode(t *testing.T) {
	in := "título ✓ você"
	if got := decode(t, Encode(in)); got != in {
		t.Fatalf("got %q", got)
	}
}

func TestShowRunsPowershellWithEncodedCommand(t *testing.T) {
	var name string
	var args []string
	toaster := Toaster{Run: func(n string, a ...string) error { name, args = n, a; return nil }}
	if err := toaster.Show("blog", "terminou"); err != nil {
		t.Fatal(err)
	}
	if name != "powershell.exe" || args[len(args)-2] != "-EncodedCommand" {
		t.Fatalf("ran %s %v", name, args)
	}
	if !strings.Contains(decode(t, args[len(args)-1]), "'blog'") {
		t.Fatal("encoded script lacks title")
	}
}
