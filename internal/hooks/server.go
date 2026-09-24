package hooks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

var ErrInUse = errors.New("another skye is listening on the socket")

type Server struct {
	srv  *http.Server
	path string
}

func Listen(path string, handle func(Event), onShow func(), onUsage func(Usage), logf func(string, ...any)) (*Server, error) {
	if conn, err := net.DialTimeout("unix", path, 500*time.Millisecond); err == nil {
		conn.Close()
		return nil, ErrInUse
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	_ = os.Remove(path)
	ln, err := net.Listen("unix", path)
	if err != nil {
		return nil, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		ln.Close()
		return nil, err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /event", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		w.WriteHeader(http.StatusNoContent)
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		ev, err := Parse(r.URL.Query().Get("t"), body)
		if err != nil {
			logf("hook ignored: %v", err)
			return
		}
		handle(ev)
	})
	mux.HandleFunc("POST /statusline", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		w.WriteHeader(http.StatusNoContent)
		if u, ok := ParseUsage(body); ok && onUsage != nil && r.URL.Query().Get("t") != "" {
			onUsage(u)
		}
	})
	mux.HandleFunc("POST /show", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
		if onShow != nil {
			onShow()
		}
	})
	s := &Server{srv: &http.Server{Handler: mux, ReadHeaderTimeout: 2 * time.Second}, path: path}
	go func() {
		err := s.srv.Serve(ln)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			logf("hook server stopped: %v", err)
		}
	}()
	return s, nil
}

func (s *Server) Close() error {
	err := s.srv.Close()
	_ = os.Remove(s.path)
	return err
}

func RequestShow(path string) error {
	c := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, "unix", path)
			},
		},
	}
	resp, err := c.Post("http://skye/show", "text/plain", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("show: status %d", resp.StatusCode)
	}
	return nil
}
