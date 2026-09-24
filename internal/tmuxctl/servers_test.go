package tmuxctl

import (
	"net"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSocketDirFollowsTmuxTmpdir(t *testing.T) {
	env := map[string]string{}
	getenv := func(k string) string { return env[k] }
	if got := SocketDir(getenv, 1000); got != "/tmp/tmux-1000" {
		t.Fatalf("default = %s", got)
	}
	env["TMUX_TMPDIR"] = "/run/x"
	if got := SocketDir(getenv, 1000); got != "/run/x/tmux-1000" {
		t.Fatalf("TMUX_TMPDIR = %s", got)
	}
}

func TestServersTellsLiveFromDeadSockets(t *testing.T) {
	dir, err := os.MkdirTemp("", "tmx")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	live, err := net.Listen("unix", filepath.Join(dir, "live"))
	if err != nil {
		t.Fatal(err)
	}
	defer live.Close()
	dead, err := net.ListenUnix("unix", &net.UnixAddr{Name: filepath.Join(dir, "dead"), Net: "unix"})
	if err != nil {
		t.Fatal(err)
	}
	dead.SetUnlinkOnClose(false)
	dead.Close()
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	got := Servers(dir)
	want := []Server{{Name: "dead", Alive: false}, {Name: "live", Alive: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("servers = %+v", got)
	}
	if Servers(filepath.Join(dir, "missing")) != nil {
		t.Fatal("servers in a missing dir")
	}
}
