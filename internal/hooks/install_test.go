package hooks

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readSettings(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m
}

func skyeCount(groups []any) int {
	n := 0
	for _, g := range groups {
		for _, h := range g.(map[string]any)["hooks"].([]any) {
			if strings.Contains(h.(map[string]any)["command"].(string), "http://skye/event") {
				n++
			}
		}
	}
	return n
}

func TestInstallCreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".claude", "settings.json")
	backup, err := Install(path)
	if err != nil {
		t.Fatal(err)
	}
	if backup != "" {
		t.Fatalf("no backup expected, got %q", backup)
	}
	hooks := readSettings(t, path)["hooks"].(map[string]any)
	for _, ev := range Events {
		if skyeCount(hooks[ev].([]any)) != 1 {
			t.Fatalf("%s missing skye hook", ev)
		}
	}
}

func TestInstallPreservesUserSettingsAndIsIdempotent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	original := `{"model":"demo","hooks":{"Stop":[{"hooks":[{"type":"command","command":"echo mine"}]}]}}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	backup, err := Install(path)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(backup); string(b) != original {
		t.Fatalf("backup = %q", b)
	}
	if _, err := Install(path); err != nil {
		t.Fatal(err)
	}
	m := readSettings(t, path)
	if m["model"] != "demo" {
		t.Fatal("other settings lost")
	}
	stop := m["hooks"].(map[string]any)["Stop"].([]any)
	if skyeCount(stop) != 1 {
		t.Fatalf("skye hook count in Stop = %d", skyeCount(stop))
	}
	data, _ := os.ReadFile(path)
	if !strings.Contains(string(data), "echo mine") {
		t.Fatal("user hook lost")
	}
}

func TestInstallRefusesInvalidJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{broken"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path); err == nil {
		t.Fatal("expected error")
	}
	if b, _ := os.ReadFile(path); string(b) != "{broken" {
		t.Fatal("file was modified")
	}
}

func TestCommandIsSafeOutsideSkye(t *testing.T) {
	c := Command()
	for _, part := range []string{`[ -n "$SKYE_TERMINAL_ID" ]`, "-m 2", "--unix-socket", "|| true"} {
		if !strings.Contains(c, part) {
			t.Errorf("command lacks %q: %s", part, c)
		}
	}
}

func statusCommand(t *testing.T, path string) string {
	t.Helper()
	sl, ok := readSettings(t, path)["statusLine"].(map[string]any)
	if !ok {
		t.Fatal("no statusLine")
	}
	return sl["command"].(string)
}

func TestInstallWrapsTheUserStatusLineOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	original := `{"statusLine":{"type":"command","command":"bash ~/.claude/statusline.sh","padding":1}}`
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path); err != nil {
		t.Fatal(err)
	}
	if _, err := Install(path); err != nil {
		t.Fatal(err)
	}
	cmd := statusCommand(t, path)
	if strings.Count(cmd, "http://skye/statusline") != 1 {
		t.Fatalf("wrapped %d times: %q", strings.Count(cmd, "http://skye/statusline"), cmd)
	}
	if got, ok := originalStatusLine(cmd); !ok || got != "bash ~/.claude/statusline.sh" {
		t.Fatalf("original = %q", got)
	}
	if readSettings(t, path)["statusLine"].(map[string]any)["padding"] != float64(1) {
		t.Fatal("statusLine fields lost")
	}
}

func TestInstallAddsARelayOnlyStatusLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if _, err := Install(path); err != nil {
		t.Fatal(err)
	}
	if got, ok := originalStatusLine(statusCommand(t, path)); !ok || got != "true" {
		t.Fatalf("original = %q", got)
	}
}
