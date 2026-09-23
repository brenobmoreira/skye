package hooks

import (
	"errors"
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

func Listen(path string, handle func(Event), logf func(string, ...any)) (*Server, error) {
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
