package web

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
)

const (
	CookieName     = "skye_token"
	cookieMaxAge   = 365 * 24 * 60 * 60
	clientQueue    = 4096
	readLimit      = 16 << 20
	writeTimeout   = 10 * time.Second
	forbiddenText  = "acesso negado: abra o endereço que o `skye web` mostrou no terminal\n"
	badOriginText  = "acesso negado: origem não permitida\n"
	loopbackHost   = "127.0.0.1"
	frameNotParsed = "requisição inválida"
	WindowsOrigin  = "http://wails.localhost"
)

type Dispatch func(method string, args []json.RawMessage) (any, error)

type Options struct {
	Token    string
	Assets   fs.FS
	Dispatch Dispatch
	Logf     func(format string, args ...any)
}

type Server struct {
	opts    Options
	ln      net.Listener
	port    int
	srv     *http.Server
	files   http.Handler
	mu      sync.Mutex
	clients map[*client]struct{}
	closed  bool
}

func Listen(port int, opts Options) (*Server, error) {
	ln, err := net.Listen("tcp4", net.JoinHostPort(loopbackHost, strconv.Itoa(port)))
	if err != nil {
		return nil, err
	}
	if opts.Logf == nil {
		opts.Logf = func(string, ...any) {}
	}
	s := &Server{
		opts:    opts,
		ln:      ln,
		port:    ln.Addr().(*net.TCPAddr).Port,
		files:   http.FileServerFS(opts.Assets),
		clients: map[*client]struct{}{},
	}
	s.srv = &http.Server{Handler: s, ReadHeaderTimeout: 5 * time.Second}
	return s, nil
}

func (s *Server) Addr() net.Addr { return s.ln.Addr() }

func (s *Server) URL() string {
	return fmt.Sprintf("http://%s:%d/?token=%s", loopbackHost, s.port, s.opts.Token)
}

func (s *Server) Serve() error {
	err := s.srv.Serve(s.ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func (s *Server) Close() error {
	s.mu.Lock()
	s.closed = true
	list := make([]*client, 0, len(s.clients))
	for c := range s.clients {
		list = append(list, c)
	}
	s.mu.Unlock()
	for _, c := range list {
		c.close()
	}
	return s.srv.Close()
}

func (s *Server) Clients() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.clients)
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "frame-ancestors 'none'")
	if r.URL.Path == "/" && r.URL.Query().Has("token") {
		s.login(w, r)
		return
	}
	if r.URL.Path == "/ws" && s.windowsApp(r) {
		s.serveSocket(w, r)
		return
	}
	if !s.hasCookie(r) {
		forbid(w, forbiddenText)
		return
	}
	if r.URL.Path == "/ws" {
		if !s.originAllowed(r.Header.Get("Origin")) {
			forbid(w, badOriginText)
			return
		}
		s.serveSocket(w, r)
		return
	}
	if strings.HasSuffix(r.URL.Path, ".webmanifest") {
		w.Header().Set("Content-Type", "application/manifest+json")
	}
	s.files.ServeHTTP(w, r)
}

func (s *Server) matches(candidate string) bool {
	return candidate != "" && subtle.ConstantTimeCompare([]byte(candidate), []byte(s.opts.Token)) == 1
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	if !s.matches(r.URL.Query().Get("token")) {
		forbid(w, forbiddenText)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    s.opts.Token,
		Path:     "/",
		MaxAge:   cookieMaxAge,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (s *Server) hasCookie(r *http.Request) bool {
	found := false
	for _, c := range r.Cookies() {
		if c.Name == CookieName && s.matches(c.Value) {
			found = true
		}
	}
	return found
}

func (s *Server) windowsApp(r *http.Request) bool {
	return r.Header.Get("Origin") == WindowsOrigin && s.matches(r.URL.Query().Get("token"))
}

func (s *Server) originAllowed(origin string) bool {
	return origin == fmt.Sprintf("http://localhost:%d", s.port) || origin == fmt.Sprintf("http://%s:%d", loopbackHost, s.port)
}

func forbid(w http.ResponseWriter, text string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	_, _ = w.Write([]byte(text))
}

type client struct {
	conn *websocket.Conn
	out  chan []byte
	done chan struct{}
	once sync.Once
}

func (c *client) close() {
	c.once.Do(func() {
		close(c.done)
		_ = c.conn.CloseNow()
	})
}

func (c *client) enqueue(msg []byte) bool {
	select {
	case c.out <- msg:
		return true
	case <-c.done:
		return false
	}
}

func (c *client) offer(msg []byte) {
	select {
	case c.out <- msg:
	default:
		c.close()
	}
}

func (c *client) writeLoop() {
	for {
		select {
		case msg := <-c.out:
			ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
			err := c.conn.Write(ctx, websocket.MessageText, msg)
			cancel()
			if err != nil {
				c.close()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (s *Server) register(c *client) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	s.clients[c] = struct{}{}
	return true
}

func (s *Server) unregister(c *client) {
	s.mu.Lock()
	delete(s.clients, c)
	s.mu.Unlock()
	c.close()
}

func (s *Server) serveSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		s.opts.Logf("web: websocket accept: %v", err)
		return
	}
	conn.SetReadLimit(readLimit)
	c := &client{conn: conn, out: make(chan []byte, clientQueue), done: make(chan struct{})}
	if !s.register(c) {
		c.close()
		return
	}
	defer s.unregister(c)
	go c.writeLoop()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-c.done
		cancel()
	}()
	for {
		typ, data, err := conn.Read(ctx)
		if err != nil {
			return
		}
		if typ != websocket.MessageText {
			continue
		}
		if !c.enqueue(s.handle(data)) {
			return
		}
	}
}

type request struct {
	ID     json.RawMessage   `json:"id"`
	Method string            `json:"method"`
	Args   []json.RawMessage `json:"args"`
}

type resultReply struct {
	ID     json.RawMessage `json:"id"`
	Result any             `json:"result"`
}

type errorReply struct {
	ID    json.RawMessage `json:"id"`
	Error string          `json:"error"`
}

type event struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func (s *Server) handle(data []byte) []byte {
	var envelope struct {
		ID json.RawMessage `json:"id"`
	}
	_ = json.Unmarshal(data, &envelope)
	id := envelope.ID
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	var req request
	if err := json.Unmarshal(data, &req); err != nil || len(req.ID) == 0 || req.Method == "" {
		return mustMarshal(errorReply{ID: id, Error: frameNotParsed})
	}
	result, err := s.call(req)
	if err != nil {
		return mustMarshal(errorReply{ID: req.ID, Error: err.Error()})
	}
	out, err := json.Marshal(resultReply{ID: req.ID, Result: result})
	if err != nil {
		return mustMarshal(errorReply{ID: req.ID, Error: err.Error()})
	}
	return out
}

func (s *Server) call(req request) (result any, err error) {
	defer func() {
		if p := recover(); p != nil {
			s.opts.Logf("web: %s panicked: %v", req.Method, p)
			result, err = nil, fmt.Errorf("%s falhou", req.Method)
		}
	}()
	return s.opts.Dispatch(req.Method, req.Args)
}

func mustMarshal(v any) []byte {
	out, _ := json.Marshal(v)
	return out
}

func (s *Server) Broadcast(name string, data any) {
	msg, err := json.Marshal(event{Event: name, Data: data})
	if err != nil {
		s.opts.Logf("web: event %s not sent: %v", name, err)
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for c := range s.clients {
		c.offer(msg)
	}
}
