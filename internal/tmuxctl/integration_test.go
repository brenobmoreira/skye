package tmuxctl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func startTest(t *testing.T) (*Client, string) {
	t.Helper()
	if _, err := exec.LookPath("tmux"); err != nil {
		t.Skip("tmux not installed")
	}
	socket := fmt.Sprintf("skye-test-%d-%d", os.Getpid(), time.Now().UnixNano())
	t.Cleanup(func() { _ = exec.Command("tmux", "-L", socket, "kill-server").Run() })
	if err := EnsureServer(socket, "/dev/null"); err != nil {
		t.Fatal(err)
	}
	if err := EnsureServer(socket, "/dev/null"); err != nil {
		t.Fatalf("second EnsureServer: %v", err)
	}
	c, err := Start(socket)
	if err != nil {
		t.Fatal(err)
	}
	return c, socket
}

func waitOutput(t *testing.T, c *Client, pane, want string) {
	t.Helper()
	var seen strings.Builder
	deadline := time.After(5 * time.Second)
	for {
		select {
		case o := <-c.Output:
			if o.Pane == pane {
				seen.Write(o.Data)
				if strings.Contains(seen.String(), want) {
					return
				}
			}
		case <-deadline:
			t.Fatalf("pane %s never printed %q; got %q", pane, want, seen.String())
		}
	}
}

func TestNewWindowRunsArgvWithEnv(t *testing.T) {
	c, _ := startTest(t)
	w, err := c.NewWindow("t1", map[string]string{"FOO": "bar baz"}, []string{"sh", "-c", "echo value=$FOO; sleep 5"})
	if err != nil {
		t.Fatal(err)
	}
	waitOutput(t, c, w.Pane, "value=bar baz")
	ws, err := c.ListWindows()
	if err != nil {
		t.Fatal(err)
	}
	if len(ws) != 1 || ws[0].SkyeID != "t1" || ws[0].ID != w.ID {
		t.Fatalf("ListWindows = %+v (placeholder must be filtered)", ws)
	}
}

func TestSendKeysPasteCaptureResize(t *testing.T) {
	c, _ := startTest(t)
	w, err := c.NewWindow("t2", nil, []string{"sh"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.SendKeys(w.Pane, []byte("echo typed-$((1+1))\r")); err != nil {
		t.Fatal(err)
	}
	waitOutput(t, c, w.Pane, "typed-2")
	if err := c.Paste(w.Pane, "echo pasted-ok"); err != nil {
		t.Fatal(err)
	}
	waitOutput(t, c, w.Pane, "pasted-ok")
	snap, err := c.Capture(w.Pane)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(snap, "typed-2") || !strings.HasPrefix(snap, "\x1b[H") {
		t.Fatalf("snapshot = %q", snap)
	}
	if err := c.Resize(w.ID, 90, 20); err != nil {
		t.Fatal(err)
	}
	lines, err := c.Command("display -p -t " + w.ID + " '#{window_width}x#{window_height}'")
	if err != nil || len(lines) == 0 || lines[0] != "90x20" {
		t.Fatalf("size = %v %v", lines, err)
	}
}

func TestErrorsAndWindowClose(t *testing.T) {
	c, _ := startTest(t)
	if _, err := c.Command("bogus-command"); err == nil {
		t.Fatal("expected error for unknown command")
	}
	w, err := c.NewWindow("t3", nil, []string{"sleep", "30"})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.KillWindow(w.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case id := <-c.Closed:
		if id != w.ID {
			t.Fatalf("closed %s, want %s", id, w.ID)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("no window-close notification")
	}
	if err := c.KillServer(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-c.Exited:
	case <-time.After(5 * time.Second):
		t.Fatal("client did not exit after kill-server")
	}
}
