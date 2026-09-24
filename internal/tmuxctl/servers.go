package tmuxctl

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// Server is a tmux socket of the user: a live server, or a file a dead one left behind.
type Server struct {
	Name  string `json:"name"`
	Alive bool   `json:"alive"`
}

// SocketDir is where tmux keeps the user's sockets.
func SocketDir(getenv func(string) string, uid int) string {
	base := getenv("TMUX_TMPDIR")
	if base == "" {
		base = "/tmp"
	}
	return filepath.Join(base, "tmux-"+strconv.Itoa(uid))
}

// Servers lists the sockets in dir; a socket is alive when something accepts a connection on it.
func Servers(dir string) []Server {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []Server
	for _, e := range entries {
		if e.Type()&os.ModeSocket == 0 {
			continue
		}
		s := Server{Name: e.Name()}
		if conn, err := net.DialTimeout("unix", filepath.Join(dir, e.Name()), 200*time.Millisecond); err == nil {
			conn.Close()
			s.Alive = true
		}
		out = append(out, s)
	}
	return out
}
