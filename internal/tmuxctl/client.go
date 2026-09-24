package tmuxctl

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	Session     = "skye"
	Placeholder = "_skye"
	keysChunk   = 512
)

var ErrExited = errors.New("tmux control client exited")

type Output struct {
	Pane string
	Data []byte
}

type Window struct {
	ID      string
	Pane    string
	SkyeID  string
	PanePID int
}

type reply struct {
	lines []string
	err   error
}

type Client struct {
	socket  string
	cmd     *exec.Cmd
	stdin   io.WriteCloser
	mu      sync.Mutex
	waiters []chan reply
	Output  chan Output
	Closed  chan string
	Exited  chan struct{}
}

func tmux(socket string, args ...string) *exec.Cmd {
	return exec.Command("tmux", append([]string{"-L", socket}, args...)...)
}

func EnsureServer(socket, conf string) error {
	if tmux(socket, "has-session", "-t", Session).Run() == nil {
		return nil
	}
	out, err := tmux(socket, "-f", conf, "new-session", "-d", "-s", Session, "-n", Placeholder, "sleep infinity").CombinedOutput()
	if err != nil {
		return fmt.Errorf("tmux new-session: %v: %s", err, out)
	}
	return nil
}

func Start(socket string) (*Client, error) {
	cmd := tmux(socket, "-C", "attach", "-t", Session)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("tmux -C attach: %w", err)
	}
	c := &Client{
		socket: socket,
		cmd:    cmd,
		stdin:  stdin,
		Output: make(chan Output, 1024),
		Closed: make(chan string, 64),
		Exited: make(chan struct{}),
	}
	go c.read(stdout)
	if _, err := c.Command("display -p ready"); err != nil {
		_ = cmd.Process.Kill()
		return nil, err
	}
	return c, nil
}

func (c *Client) read(r io.Reader) {
	defer close(c.Exited)
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64*1024), 16*1024*1024)
	var (
		inBlock bool
		ours    bool
		number  string
		lines   []string
	)
	for sc.Scan() {
		text := sc.Text()
		l := ParseLine(text)
		if inBlock {
			if (l.Kind == KindEnd || l.Kind == KindError) && l.Number == number {
				inBlock = false
				if ours {
					c.deliver(lines, l.Kind == KindError)
				}
				continue
			}
			lines = append(lines, text)
			continue
		}
		switch l.Kind {
		case KindBegin:
			inBlock, ours, number, lines = true, l.Flags == "1", l.Number, nil
		case KindOutput:
			c.Output <- Output{Pane: l.Pane, Data: l.Data}
		case KindWindowClose:
			c.Closed <- l.Window
		}
	}
}

func (c *Client) deliver(lines []string, failed bool) {
	c.mu.Lock()
	if len(c.waiters) == 0 {
		c.mu.Unlock()
		return
	}
	ch := c.waiters[0]
	c.waiters = c.waiters[1:]
	c.mu.Unlock()
	if failed {
		ch <- reply{err: fmt.Errorf("tmux: %s", strings.Join(lines, " "))}
		return
	}
	ch <- reply{lines: lines}
}

func (c *Client) Command(line string) ([]string, error) {
	ch := make(chan reply, 1)
	c.mu.Lock()
	c.waiters = append(c.waiters, ch)
	_, err := io.WriteString(c.stdin, line+"\n")
	c.mu.Unlock()
	if err != nil {
		return nil, fmt.Errorf("tmux: %w", err)
	}
	select {
	case r := <-ch:
		return r.lines, r.err
	case <-c.Exited:
		return nil, ErrExited
	case <-time.After(5 * time.Second):
		return nil, fmt.Errorf("tmux: no reply to %q", line)
	}
}

func Quote(s string) (string, error) {
	if strings.ContainsAny(s, "'\n") {
		return "", fmt.Errorf("tmux: unsupported character in %q", s)
	}
	return "'" + s + "'", nil
}

func (c *Client) NewWindow(skyeID string, env map[string]string, argv []string) (Window, error) {
	id, err := Quote(skyeID)
	if err != nil {
		return Window{}, err
	}
	parts := []string{"new-window", "-d", "-P", "-F", "'#{window_id} #{pane_id}'", "-t", Session + ":"}
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		q, err := Quote(k + "=" + env[k])
		if err != nil {
			return Window{}, err
		}
		parts = append(parts, "-e", q)
	}
	for _, a := range argv {
		q, err := Quote(a)
		if err != nil {
			return Window{}, err
		}
		parts = append(parts, q)
	}
	lines, err := c.Command(strings.Join(parts, " "))
	if err != nil {
		return Window{}, err
	}
	if len(lines) == 0 {
		return Window{}, errors.New("tmux: new-window returned nothing")
	}
	fields := strings.Fields(lines[0])
	if len(fields) != 2 {
		return Window{}, fmt.Errorf("tmux: unexpected new-window reply %q", lines[0])
	}
	w := Window{ID: fields[0], Pane: fields[1], SkyeID: skyeID}
	if _, err := c.Command(fmt.Sprintf("set-option -w -t %s @skye_id %s", w.ID, id)); err != nil {
		_, _ = c.Command("kill-window -t " + w.ID)
		return Window{}, err
	}
	return w, nil
}

func (c *Client) ListWindows() ([]Window, error) {
	// @skye_id goes last: the placeholder window has none, so its line is one field short.
	lines, err := c.Command("list-windows -t " + Session + " -F '#{window_id} #{pane_id} #{pane_pid} #{@skye_id}'")
	if err != nil {
		return nil, err
	}
	windows := []Window{}
	for _, line := range lines {
		f := strings.Fields(line)
		if len(f) < 4 {
			continue
		}
		pid, _ := strconv.Atoi(f[2])
		windows = append(windows, Window{ID: f[0], Pane: f[1], PanePID: pid, SkyeID: f[3]})
	}
	return windows, nil
}

func (c *Client) SendKeys(pane string, data []byte) error {
	for len(data) > 0 {
		n := min(len(data), keysChunk)
		h := hex.EncodeToString(data[:n])
		pairs := make([]string, 0, n)
		for i := 0; i < len(h); i += 2 {
			pairs = append(pairs, h[i:i+2])
		}
		if _, err := c.Command(fmt.Sprintf("send-keys -t %s -H %s", pane, strings.Join(pairs, " "))); err != nil {
			return err
		}
		data = data[n:]
	}
	return nil
}

func (c *Client) Capture(pane string) (string, error) {
	lines, err := c.Command("capture-pane -p -e -t " + pane)
	if err != nil {
		return "", err
	}
	cur, err := c.Command("display -p -t " + pane + " '#{cursor_x} #{cursor_y}'")
	if err != nil {
		return "", err
	}
	var x, y int
	if len(cur) > 0 {
		fmt.Sscanf(cur[0], "%d %d", &x, &y)
	}
	return RenderSnapshot(lines, x, y), nil
}

func RenderSnapshot(lines []string, x, y int) string {
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return "\x1b[H\x1b[2J" + strings.Join(lines[:end], "\r\n") + fmt.Sprintf("\x1b[%d;%dH", y+1, x+1)
}

func (c *Client) Resize(window string, cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return fmt.Errorf("tmux: invalid size %dx%d", cols, rows)
	}
	_, err := c.Command(fmt.Sprintf("refresh-client -C %s:%dx%d", window, cols, rows))
	return err
}

func (c *Client) KillWindow(window string) error {
	_, err := c.Command("kill-window -t " + window)
	return err
}

func (c *Client) KillServer() error {
	_, err := c.Command("kill-server")
	if errors.Is(err, ErrExited) {
		return nil
	}
	return err
}

func (c *Client) Close() error {
	_ = c.stdin.Close()
	return c.cmd.Wait()
}
