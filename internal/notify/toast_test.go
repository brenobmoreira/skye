package notify

import (
	"encoding/base64"
	"regexp"
	"strings"
	"testing"
)

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

func TestShowPipesTheScriptIntoPowershell(t *testing.T) {
	var stdin, name string
	var args []string
	toaster := Toaster{Run: func(in, n string, a ...string) error { stdin, name, args = in, n, a; return nil }}
	if err := toaster.Show("blog ✓", "terminou"); err != nil {
		t.Fatal(err)
	}
	if name != "powershell.exe" || strings.Join(args, " ") != "-NoProfile -NonInteractive -Command -" {
		t.Fatalf("ran %s %v", name, args)
	}
	for _, r := range stdin {
		if r > 127 {
			t.Fatalf("non-ascii %q reaches powershell's stdin", r)
		}
	}
	if !strings.HasSuffix(stdin, "\n") {
		t.Fatal("last line of the script is not terminated")
	}
	if got := embeddedTexts(t, stdin); len(got) != 2 || got[0] != "blog ✓" || got[1] != "terminou" {
		t.Fatalf("embedded = %q", got)
	}
}
