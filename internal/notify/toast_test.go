package notify

import (
	"encoding/base64"
	"encoding/binary"
	"regexp"
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

var embedded = regexp.MustCompile(`FromBase64String\('([A-Za-z0-9+/=]*)'\)`)

func embeddedTexts(t *testing.T, script string) []string {
	t.Helper()
	var out []string
	for _, m := range embedded.FindAllStringSubmatch(script, -1) {
		raw, err := base64.StdEncoding.DecodeString(m[1])
		if err != nil {
			t.Fatal(err)
		}
		out = append(out, string(raw))
	}
	return out
}

func TestScriptCarriesTextAsBase64(t *testing.T) {
	s := Script("it's você", "esperando você")
	got := embeddedTexts(t, s)
	if len(got) != 2 || got[0] != "it's você" || got[1] != "esperando você" {
		t.Fatalf("embedded = %q in %s", got, s)
	}
	if strings.Contains(s, "você") {
		t.Fatalf("raw text leaked into script: %s", s)
	}
}

func TestScriptDoesNotInterpolateCurlyQuotes(t *testing.T) {
	title := "x’)) | Out-Null; calc; ((’"
	s := Script(title, "b")
	if strings.Contains(s, title) || strings.ContainsRune(s, '’') || strings.Contains(s, "calc") {
		t.Fatalf("script contains raw title: %s", s)
	}
	if got := embeddedTexts(t, s); len(got) != 2 || got[0] != title || got[1] != "b" {
		t.Fatalf("embedded = %q", got)
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
	if got := embeddedTexts(t, decode(t, args[len(args)-1])); len(got) != 2 || got[0] != "blog" {
		t.Fatal("encoded script lacks title")
	}
}
