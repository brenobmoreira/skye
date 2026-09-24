package winshell

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

const validToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func env(values map[string]string) func(string) string {
	return func(k string) string { return values[k] }
}

func TestPort(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  int
	}{
		{"unset", "", DefaultPort},
		{"valid", "9000", 9000},
		{"padded", " 9001 ", 9001},
		{"garbage", "abc", DefaultPort},
		{"too low", "80", DefaultPort},
		{"too high", "70000", DefaultPort},
		{"lower bound", "1024", 1024},
		{"upper bound", "65535", 65535},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Port(env(map[string]string{"SKYE_WEB_PORT": c.value})); got != c.want {
				t.Fatalf("Port(%q) = %d, want %d", c.value, got, c.want)
			}
		})
	}
}

func TestProbe(t *testing.T) {
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				return
			}
			c.Close()
		}
	}()
	if !Probe(port, time.Second) {
		t.Fatal("probe false against a listener")
	}
	ln.Close()
	if Probe(port, 300*time.Millisecond) {
		t.Fatal("probe true against a closed port")
	}
}

func TestStartArgs(t *testing.T) {
	name, args := StartArgs()
	if name != "wsl.exe" {
		t.Fatalf("name = %q", name)
	}
	if want := []string{"--", "bash", "-lc", "skye web --no-open"}; !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %q, want %q", args, want)
	}
}

type fakeClock struct {
	slept []time.Duration
}

func (f *fakeClock) sleep(d time.Duration) { f.slept = append(f.slept, d) }

func (f *fakeClock) total() time.Duration {
	var sum time.Duration
	for _, d := range f.slept {
		sum += d
	}
	return sum
}

func probeAfter(n int) (func(int, time.Duration) bool, *int) {
	calls := 0
	return func(int, time.Duration) bool {
		calls++
		return calls > n
	}, &calls
}

func TestWaitReadySucceedsAfterSomeProbes(t *testing.T) {
	clock := &fakeClock{}
	probe, calls := probeAfter(3)
	if err := WaitReady(7810, 15*time.Second, probe, clock.sleep); err != nil {
		t.Fatal(err)
	}
	if *calls != 4 {
		t.Fatalf("probes = %d, want 4", *calls)
	}
	if len(clock.slept) != 3 {
		t.Fatalf("sleeps = %v", clock.slept)
	}
	for _, d := range clock.slept {
		if d != 250*time.Millisecond {
			t.Fatalf("sleep %v, want 250ms", d)
		}
	}
}

func TestWaitReadyFailsAtTheDeadline(t *testing.T) {
	clock := &fakeClock{}
	never := func(int, time.Duration) bool { return false }
	err := WaitReady(7811, 15*time.Second, never, clock.sleep)
	if err == nil {
		t.Fatal("no error when the server never answers")
	}
	if !strings.Contains(err.Error(), "7811") {
		t.Fatalf("error %q does not name the port", err)
	}
	if got := clock.total(); got < 15*time.Second || got > 15*time.Second+250*time.Millisecond {
		t.Fatalf("waited %v, want about 15s", got)
	}
}

func runnerReturning(out string, err error, seen *[][]string) Runner {
	return func(name string, args ...string) ([]byte, error) {
		if seen != nil {
			*seen = append(*seen, append([]string{name}, args...))
		}
		return []byte(out), err
	}
}

func TestReadTokenAcceptsPaddedToken(t *testing.T) {
	var seen [][]string
	tok, err := ReadToken(runnerReturning("\n  "+validToken+"\r\n", nil, &seen))
	if err != nil {
		t.Fatal(err)
	}
	if tok != validToken {
		t.Fatalf("token = %q", tok)
	}
	want := [][]string{{"wsl.exe", "--", "bash", "-lc", "cat ~/.config/skye/web-token"}}
	if !reflect.DeepEqual(seen, want) {
		t.Fatalf("ran %q, want %q", seen, want)
	}
}

func TestReadTokenRejectsInvalidContent(t *testing.T) {
	for _, out := range []string{
		"",
		validToken[:63],
		validToken + "0",
		strings.ToUpper(validToken),
		"cat: /home/x/.config/skye/web-token: No such file or directory",
		strings.Repeat("g", 64),
	} {
		_, err := ReadToken(runnerReturning(out, nil, nil))
		if err == nil {
			t.Fatalf("accepted %q", out)
		}
		if !strings.Contains(err.Error(), "cat ~/.config/skye/web-token") {
			t.Fatalf("error %q does not name the command", err)
		}
	}
}

func TestReadTokenRunnerErrorCarriesOutput(t *testing.T) {
	_, err := ReadToken(runnerReturning("WSL não tem nenhuma distribuição instalada", errors.New("exit status 1"), nil))
	if err == nil {
		t.Fatal("no error")
	}
	for _, part := range []string{"WSL não tem nenhuma distribuição instalada", "exit status 1", `wsl.exe -- bash -lc "cat ~/.config/skye/web-token"`} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error %q lacks %q", err, part)
		}
	}
}

type fakeStarter struct {
	calls [][]string
	err   error
}

func (f *fakeStarter) start(name string, args ...string) error {
	f.calls = append(f.calls, append([]string{name}, args...))
	return f.err
}

func TestEnsureWhenAlreadyUpNeverStarts(t *testing.T) {
	starter := &fakeStarter{}
	clock := &fakeClock{}
	up := func(int, time.Duration) bool { return true }
	ep, err := Ensure(7810, runnerReturning(validToken, nil, nil), starter.start, up, clock.sleep)
	if err != nil {
		t.Fatal(err)
	}
	if len(starter.calls) != 0 {
		t.Fatalf("started %q while the server was up", starter.calls)
	}
	if ep != (Endpoint{URL: "ws://127.0.0.1:7810/ws", Token: validToken}) {
		t.Fatalf("endpoint = %+v", ep)
	}
}

func TestEnsureStartsOnceWhenDown(t *testing.T) {
	starter := &fakeStarter{}
	clock := &fakeClock{}
	probe, _ := probeAfter(2)
	ep, err := Ensure(9000, runnerReturning(validToken, nil, nil), starter.start, probe, clock.sleep)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"wsl.exe", "--", "bash", "-lc", "skye web --no-open"}}
	if !reflect.DeepEqual(starter.calls, want) {
		t.Fatalf("started %q, want %q", starter.calls, want)
	}
	if ep != (Endpoint{URL: "ws://127.0.0.1:9000/ws", Token: validToken}) {
		t.Fatalf("endpoint = %+v", ep)
	}
}

func TestEnsureSurfacesStartError(t *testing.T) {
	starter := &fakeStarter{err: errors.New("executable file not found in %PATH%")}
	clock := &fakeClock{}
	down := func(int, time.Duration) bool { return false }
	_, err := Ensure(7810, runnerReturning(validToken, nil, nil), starter.start, down, clock.sleep)
	if err == nil {
		t.Fatal("no error")
	}
	for _, part := range []string{"executable file not found", `wsl.exe -- bash -lc "skye web --no-open"`} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error %q lacks %q", err, part)
		}
	}
	if len(clock.slept) != 0 {
		t.Fatalf("waited after a failed start: %v", clock.slept)
	}
}

func TestEnsureSurfacesNeverReady(t *testing.T) {
	starter := &fakeStarter{}
	clock := &fakeClock{}
	down := func(int, time.Duration) bool { return false }
	ran := false
	run := func(string, ...string) ([]byte, error) { ran = true; return []byte(validToken), nil }
	_, err := Ensure(7810, run, starter.start, down, clock.sleep)
	if err == nil {
		t.Fatal("no error")
	}
	for _, part := range []string{"7810", `wsl.exe -- bash -lc "skye web --no-open"`, "~/.local/bin/skye"} {
		if !strings.Contains(err.Error(), part) {
			t.Fatalf("error %q lacks %q", err, part)
		}
	}
	if ran {
		t.Fatal("read the token from a server that never answered")
	}
}

func TestEnsureSurfacesTokenError(t *testing.T) {
	up := func(int, time.Duration) bool { return true }
	_, err := Ensure(7810, runnerReturning("lixo", nil, nil), (&fakeStarter{}).start, up, (&fakeClock{}).sleep)
	if err == nil || !strings.Contains(err.Error(), "lixo") {
		t.Fatalf("err = %v", err)
	}
}
