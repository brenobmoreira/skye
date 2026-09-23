package hooks

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func client(path string) *http.Client {
	return &http.Client{Transport: &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", path)
		},
	}}
}

func TestServerDeliversEventsAndAnswers204(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")
	got := make(chan Event, 1)
	srv, err := Listen(path, func(e Event) { got <- e }, nil, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("socket mode %v", info.Mode().Perm())
	}

	resp, err := client(path).Post("http://skye/event?t=t1", "application/json", strings.NewReader(`{"hook_event_name":"Stop","session_id":"s1"}`))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	select {
	case e := <-got:
		if e.Terminal != "t1" || e.Name != "Stop" {
			t.Fatalf("got %+v", e)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("handler not called")
	}

	resp, err = client(path).Post("http://skye/event", "application/json", strings.NewReader(`garbage`))
	if err != nil || resp.StatusCode != http.StatusNoContent {
		t.Fatalf("invalid payload: %v %v", resp, err)
	}
}

func TestListenDetectsRunningInstanceAndStaleSocket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")
	srv, err := Listen(path, func(Event) {}, nil, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Listen(path, func(Event) {}, nil, t.Logf); !errors.Is(err, ErrInUse) {
		t.Fatalf("second Listen: %v", err)
	}
	srv.Close()

	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	srv2, err := Listen(path, func(Event) {}, nil, t.Logf)
	if err != nil {
		t.Fatalf("stale socket: %v", err)
	}
	srv2.Close()
}

func TestServerShowsWindowOnRequest(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")
	shown := make(chan struct{}, 1)
	srv, err := Listen(path, func(Event) {}, func() { shown <- struct{}{} }, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()

	resp, err := client(path).Post("http://skye/show", "text/plain", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("status %d", resp.StatusCode)
	}
	select {
	case <-shown:
	case <-time.After(2 * time.Second):
		t.Fatal("onShow not called")
	}
}

func TestRequestShow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.sock")
	if err := RequestShow(path); err == nil {
		t.Fatal("RequestShow succeeded with nothing listening")
	}
	shown := make(chan struct{}, 1)
	srv, err := Listen(path, func(Event) {}, func() { shown <- struct{}{} }, t.Logf)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	if err := RequestShow(path); err != nil {
		t.Fatalf("RequestShow: %v", err)
	}
	select {
	case <-shown:
	case <-time.After(2 * time.Second):
		t.Fatal("onShow not called")
	}
}
