package resume

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetTitleAppendsCustomTitleToTranscript(t *testing.T) {
	projects := t.TempDir()
	dir := filepath.Join(projects, "-home-demo-blog")
	os.MkdirAll(dir, 0o700)
	transcript := filepath.Join(dir, "s1.jsonl")
	os.WriteFile(transcript, []byte(`{"type":"user"}`+"\n"), 0o600)
	if err := SetTitle(projects, "s1", `api "v2"`); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(transcript)
	want := `{"type":"user"}` + "\n" + `{"type":"custom-title","customTitle":"api \"v2\"","sessionId":"s1"}` + "\n"
	if string(data) != want {
		t.Fatalf("transcript =\n%s", data)
	}
}

func TestSetTitleSkipsSessionWithoutTranscript(t *testing.T) {
	projects := t.TempDir()
	if err := SetTitle(projects, "s1", "api"); err != nil {
		t.Fatal(err)
	}
	if entries, _ := os.ReadDir(projects); len(entries) != 0 {
		t.Fatalf("created %v", entries)
	}
}

func TestSetTitleRejectsUnsafeSessionID(t *testing.T) {
	if err := SetTitle(t.TempDir(), "../x", "api"); err == nil || !strings.Contains(err.Error(), "invalid") {
		t.Fatalf("err = %v", err)
	}
}
