package winshell

import (
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Runner func(name string, args ...string) ([]byte, error)

type Starter func(name string, args ...string) error

const (
	DefaultPort  = 7810
	ReadyTimeout = 15 * time.Second
	ProbeTimeout = 300 * time.Millisecond
	pollInterval = 250 * time.Millisecond
	wsl          = "wsl.exe"
	tokenCommand = `cat "${XDG_CONFIG_HOME:-$HOME/.config}/skye/web-token"`
	errorOutput  = 200
	startCommand = "skye web --no-open"
)

var (
	tokenPattern  = regexp.MustCompile(`^[0-9a-f]{64}$`)
	secretPattern = regexp.MustCompile(`[0-9a-fA-F]{64}`)
	gaveUpLines   = []string{"a skye já está aberta", "outra skye já está rodando"}
)

func Redact(s string) string {
	return secretPattern.ReplaceAllString(s, "<oculto>")
}

func GaveUp(log string) (string, bool) {
	for _, line := range gaveUpLines {
		if strings.Contains(log, line) {
			return line, true
		}
	}
	return "", false
}

func shortOutput(s string) string {
	r := []rune(Redact(s))
	if len(r) > errorOutput {
		return string(r[:errorOutput]) + "…"
	}
	return string(r)
}

type Endpoint struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

func Port(getenv func(string) string) int {
	port, err := strconv.Atoi(strings.TrimSpace(getenv("SKYE_WEB_PORT")))
	if err != nil || port < 1024 || port > 65535 {
		return DefaultPort
	}
	return port
}

func Probe(port int, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp4", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), timeout)
	if err != nil {
		return false
	}
	_ = conn.Close()
	return true
}

func StartArgs() (string, []string) {
	return wsl, []string{"-e", "bash", "-lc", startCommand}
}

func tokenArgs() (string, []string) {
	return wsl, []string{"-e", "sh", "-c", tokenCommand}
}

func commandLine(name string, args []string) string {
	parts := []string{name}
	for _, a := range args {
		if strings.ContainsAny(a, " \t") {
			a = "'" + a + "'"
		}
		parts = append(parts, a)
	}
	return strings.Join(parts, " ")
}

func WaitReady(port int, deadline time.Duration, probe func(int, time.Duration) bool, stopped func() (string, bool), sleep func(time.Duration)) error {
	began := time.Now()
	var waited time.Duration
	for {
		if probe(port, ProbeTimeout) {
			return nil
		}
		if why, ok := stopped(); ok {
			return gaveUpError{port: port, why: why}
		}
		if waited >= deadline || time.Since(began) >= deadline {
			return fmt.Errorf("o servidor da skye não respondeu na porta %d em %s", port, deadline)
		}
		sleep(pollInterval)
		waited += pollInterval
	}
}

func ReadToken(run Runner) (string, error) {
	name, args := tokenArgs()
	line := commandLine(name, args)
	out, err := run(name, args...)
	text := strings.TrimSpace(string(out))
	if err != nil {
		return "", fmt.Errorf("não consegui ler o token da skye: %s falhou (%v)\n%s", line, err, shortOutput(text))
	}
	if !tokenPattern.MatchString(text) {
		return "", fmt.Errorf("token da skye inválido: %s devolveu\n%s", line, shortOutput(text))
	}
	return text, nil
}

type gaveUpError struct {
	port int
	why  string
}

func (e gaveUpError) Error() string {
	return fmt.Sprintf("o servidor da skye não subiu na porta %d: %s", e.port, e.why)
}

func Ensure(port int, run Runner, start Starter, probe func(int, time.Duration) bool, stopped func() (string, bool), sleep func(time.Duration)) (Endpoint, error) {
	if !probe(port, ProbeTimeout) {
		name, args := StartArgs()
		line := commandLine(name, args)
		if err := start(name, args...); err != nil {
			return Endpoint{}, fmt.Errorf("não consegui iniciar a skye no WSL: %s falhou\n%v", line, err)
		}
		if err := WaitReady(port, ReadyTimeout, probe, stopped, sleep); err != nil {
			if errors.As(err, new(gaveUpError)) {
				return Endpoint{}, fmt.Errorf("%w (%s)", err, line)
			}
			return Endpoint{}, fmt.Errorf("%w depois de %s\nconfira se a skye está instalada no WSL: make build && install -m 755 skye ~/.local/bin/skye", err, line)
		}
	}
	token, err := ReadToken(run)
	if err != nil {
		return Endpoint{}, err
	}
	return Endpoint{URL: fmt.Sprintf("ws://127.0.0.1:%d/ws", port), Token: token}, nil
}
