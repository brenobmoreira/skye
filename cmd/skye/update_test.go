package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestWebServersFindsOnlySkyeWeb(t *testing.T) {
	proc := t.TempDir()
	put := func(pid, cmdline string) {
		os.MkdirAll(filepath.Join(proc, pid), 0o755)
		os.WriteFile(filepath.Join(proc, pid, "cmdline"), []byte(cmdline), 0o644)
	}
	put("10", "/home/demo/.local/bin/skye\x00web\x00--no-open\x00")
	put("11", "skye\x00update\x00")
	put("12", "bash\x00-c\x00skye web\x00")
	put("13", "skye\x00web\x00")
	put("self", "")
	got := webServers(proc, 13)
	if !reflect.DeepEqual(got, []int{10}) {
		t.Fatalf("got %v", got)
	}
}
