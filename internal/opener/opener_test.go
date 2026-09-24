package opener

import (
	"reflect"
	"testing"
)

func TestCommandUsesTheWindowsBrowserInsideWSL(t *testing.T) {
	name, args, err := Command("https://example.com/a?b=1&c=2", true)
	if err != nil || name != "explorer.exe" || !reflect.DeepEqual(args, []string{"https://example.com/a?b=1&c=2"}) {
		t.Fatalf("got %q %q %v", name, args, err)
	}
}

func TestCommandUsesXdgOpenOutsideWSL(t *testing.T) {
	name, args, err := Command("http://localhost:3000", false)
	if err != nil || name != "xdg-open" || !reflect.DeepEqual(args, []string{"http://localhost:3000"}) {
		t.Fatalf("got %q %q %v", name, args, err)
	}
}

func TestCommandRefusesAnythingButWebLinks(t *testing.T) {
	for _, u := range []string{"file:///etc/passwd", "javascript:alert(1)", "/home/demo", "ftp://x", "https://", "https://x/\n", "-https://x"} {
		if _, _, err := Command(u, true); err == nil {
			t.Errorf("Command(%q) accepted", u)
		}
	}
}

func TestInWSL(t *testing.T) {
	if !InWSL(func(string) string { return "Ubuntu" }, func() string { return "" }) {
		t.Fatal("WSL_DISTRO_NAME ignored")
	}
	if !InWSL(func(string) string { return "" }, func() string { return "Linux version 6.6 (microsoft-standard-WSL2)" }) {
		t.Fatal("/proc/version ignored")
	}
	if InWSL(func(string) string { return "" }, func() string { return "Linux version 6.6 (debian)" }) {
		t.Fatal("plain linux taken as WSL")
	}
}
