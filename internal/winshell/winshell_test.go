package winshell

import (
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	validToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	tokenLine  = `wsl.exe -e sh -c 'cat "${XDG_CONFIG_HOME:-$HOME/.config}/skye/web-token"'`
	startLine  = `wsl.exe -e bash -lc 'skye web --no-open'`
)

func running() (string, bool) { return "", false }

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
	if want := []string{"-e", "bash", "-lc", "skye web --no-open"}; !reflect.DeepEqual(args, want) {
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
	if err := WaitReady(7810, 15*time.Second, probe, running, clock.sleep); err != nil {
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
	err := WaitReady(7811, 15*time.Second, never, running, clock.sleep)
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
	want := [][]string{{"wsl.exe", "-e", "sh", "-c", `cat "${XDG_CONFIG_HOME:-$HOME/.config}/skye/web-token"`}}
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
		if !strings.Contains(err.Error(), tokenLine) {
			t.Fatalf("error %q does not name the command", err)
		}
	}
}

func TestReadTokenRunnerErrorCarriesOutput(t *testing.T) {
	_, err := ReadToken(runnerReturning("WSL não tem nenhuma distribuição instalada", errors.New("exit status 1"), nil))
	if err == nil {
		t.Fatal("no error")
	}
	for _, part := range []string{"WSL não tem nenhuma distribuição instalada", "exit status 1", tokenLine} {
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
	ep, err := Ensure(7810, runnerReturning(validToken, nil, nil), starter.start, up, running, clock.sleep)
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
	ep, err := Ensure(9000, runnerReturning(validToken, nil, nil), starter.start, probe, running, clock.sleep)
	if err != nil {
		t.Fatal(err)
	}
	want := [][]string{{"wsl.exe", "-e", "bash", "-lc", "skye web --no-open"}}
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
	_, err := Ensure(7810, runnerReturning(validToken, nil, nil), starter.start, down, running, clock.sleep)
	if err == nil {
		t.Fatal("no error")
	}
	for _, part := range []string{"executable file not found", startLine} {
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
	_, err := Ensure(7810, run, starter.start, down, running, clock.sleep)
	if err == nil {
		t.Fatal("no error")
	}
	for _, part := range []string{"7810", startLine, "~/.local/bin/skye"} {
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
	_, err := Ensure(7810, runnerReturning("lixo", nil, nil), (&fakeStarter{}).start, up, running, (&fakeClock{}).sleep)
	if err == nil || !strings.Contains(err.Error(), "lixo") {
		t.Fatalf("err = %v", err)
	}
}

func TestWaitReadyStopsEarlyWhenTheServerGaveUp(t *testing.T) {
	clock := &fakeClock{}
	never := func(int, time.Duration) bool { return false }
	checks := 0
	stopped := func() (string, bool) {
		checks++
		if checks < 3 {
			return "", false
		}
		return "outra skye já está rodando", true
	}
	err := WaitReady(7810, 15*time.Second, never, stopped, clock.sleep)
	if err == nil || !strings.Contains(err.Error(), "outra skye já está rodando") {
		t.Fatalf("err = %v", err)
	}
	if clock.total() > time.Second {
		t.Fatalf("waited %v before giving up", clock.total())
	}
}

func TestEnsureFailsFastWhenTheStartedServerGaveUp(t *testing.T) {
	down := func(int, time.Duration) bool { return false }
	gaveUp := func() (string, bool) { return "a skye já está aberta", true }
	clock := &fakeClock{}
	_, err := Ensure(7810, runnerReturning(validToken, nil, nil), (&fakeStarter{}).start, down, gaveUp, clock.sleep)
	if err == nil || !strings.Contains(err.Error(), "a skye já está aberta") || !strings.Contains(err.Error(), startLine) {
		t.Fatalf("err = %v", err)
	}
	if len(clock.slept) != 0 {
		t.Fatalf("slept %v", clock.slept)
	}
}

func TestGaveUpFindsTheOtherSkyeLines(t *testing.T) {
	for _, c := range []struct {
		log  string
		want string
		ok   bool
	}{
		{"", "", false},
		{"skye no navegador: http://127.0.0.1:7810/?token=x\n", "", false},
		{"a skye já está aberta\n", "a skye já está aberta", true},
		{"2026/09/24 boot\noutra skye já está rodando\n", "outra skye já está rodando", true},
	} {
		got, ok := GaveUp(c.log)
		if got != c.want || ok != c.ok {
			t.Fatalf("GaveUp(%q) = %q, %v", c.log, got, ok)
		}
	}
}

func TestRedactHidesTokens(t *testing.T) {
	in := "skye no navegador: http://127.0.0.1:7810/?token=" + validToken + "\noutra linha " + strings.ToUpper(validToken)
	out := Redact(in)
	if strings.Contains(out, validToken) || strings.Contains(out, strings.ToUpper(validToken)) {
		t.Fatalf("token left in %q", out)
	}
	if !strings.Contains(out, "?token=<oculto>") || !strings.Contains(out, "outra linha") {
		t.Fatalf("redacted = %q", out)
	}
}

func TestReadTokenErrorHidesTokensAndTruncates(t *testing.T) {
	noisy := "bem-vindo " + validToken + "\n" + strings.Repeat("x", 500)
	_, err := ReadToken(runnerReturning(noisy, nil, nil))
	if err == nil {
		t.Fatal("accepted noisy output")
	}
	if strings.Contains(err.Error(), validToken) {
		t.Fatalf("error leaks the token: %q", err)
	}
	if strings.Contains(err.Error(), strings.Repeat("x", 201)) {
		t.Fatalf("error not truncated: %d chars", len(err.Error()))
	}
	if !strings.Contains(err.Error(), "bem-vindo <oculto>") {
		t.Fatalf("error = %q", err)
	}
	_, err = ReadToken(runnerReturning(validToken+" extra", errors.New("exit status 1"), nil))
	if err == nil || strings.Contains(err.Error(), validToken) {
		t.Fatalf("runner error leaks the token: %v", err)
	}
}

func TestEnsureGaveUpDoesNotBlameTheInstall(t *testing.T) {
	down := func(int, time.Duration) bool { return false }
	gaveUp := func() (string, bool) { return "outra skye já está rodando", true }
	_, err := Ensure(7810, runnerReturning(validToken, nil, nil), (&fakeStarter{}).start, down, gaveUp, (&fakeClock{}).sleep)
	if err == nil || strings.Contains(err.Error(), "~/.local/bin/skye") {
		t.Fatalf("err = %v", err)
	}
}
