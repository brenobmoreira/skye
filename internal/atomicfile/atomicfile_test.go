package atomicfile

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteReplacesTheFileWhole(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "missing", "dir")
	file := filepath.Join(dir, "notes.json")
	if err := Write(file, []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Write(file, []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "b" {
		t.Fatalf("content = %q", data)
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Fatalf("leftovers in dir: %v", entries)
	}
}
