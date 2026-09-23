package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"
)

const testToken = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestLoadOrCreateTokenCreatesPrivateFileAndReusesIt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skye", "web-token")
	first, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(first) != 64 {
		t.Fatalf("token %q is not 32 hex bytes", first)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
	second, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if second != first {
		t.Fatalf("token changed across loads: %q then %q", first, second)
	}
}

func TestLoadOrCreateTokenReplacesInvalidContent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "web-token")
	if err := os.WriteFile(path, []byte("short\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tok, err := LoadOrCreateToken(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(tok) != 64 || tok == "short" {
		t.Fatalf("token = %q", tok)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v, want 0600", info.Mode().Perm())
	}
}

type harness struct {
	srv  *Server
	base string
	port int
}

func echoDispatch(method string, args []json.RawMessage) (any, error) {
	switch method {
	case "Echo":
		var s string
		if len(args) != 1 || json.Unmarshal(args[0], &s) != nil {
			return nil, errors.New("bad args")
		}
		return s, nil
	case "Panic":
		panic("boom")
	}
	return nil, fmt.Errorf("unknown method %q", method)
}

func start(t *testing.T) harness {
	t.Helper()
	s, err := Listen(0, Options{
		Token:    testToken,
		Assets:   fstest.MapFS{"index.html": {Data: []byte("<html>skye</html>")}, "app.js": {Data: []byte("x")}, "manifest.webmanifest": {Data: []byte(`{"name":"skye"}`)}},
		Dispatch: echoDispatch,
		Logf:     t.Logf,
	})
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = s.Serve() }()
	t.Cleanup(func() { _ = s.Close() })
	port := s.Addr().(*net.TCPAddr).Port
	return harness{srv: s, base: fmt.Sprintf("http://127.0.0.1:%d", port), port: port}
}

func noRedirect() *http.Client {
	return &http.Client{
		Timeout:       5 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
}

func get(t *testing.T, url string, cookie bool) *http.Response {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url, nil)
	if cookie {
		req.AddCookie(&http.Cookie{Name: CookieName, Value: testToken})
	}
	resp, err := noRedirect().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { resp.Body.Close() })
	return resp
}

func TestListenBindsOnlyLoopback(t *testing.T) {
	h := start(t)
	addr := h.srv.Addr().(*net.TCPAddr)
	if !addr.IP.Equal(net.IPv4(127, 0, 0, 1)) {
		t.Fatalf("listening on %v, want 127.0.0.1", addr)
	}
	if got := h.srv.URL(); got != fmt.Sprintf("http://localhost:%d/?token=%s", h.port, testToken) {
		t.Fatalf("url = %q", got)
	}
}

func TestLoginWithRightTokenSetsCookieAndRedirects(t *testing.T) {
	h := start(t)
	resp := get(t, h.base+"/?token="+testToken, false)
	if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/" {
		t.Fatalf("status %d location %q", resp.StatusCode, resp.Header.Get("Location"))
	}
	var c *http.Cookie
	for _, k := range resp.Cookies() {
		if k.Name == CookieName {
			c = k
		}
	}
	if c == nil {
		t.Fatal("no cookie set")
	}
	if c.Value != testToken || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.MaxAge != 365*24*60*60 {
		t.Fatalf("cookie = %+v", c)
	}
}

func TestLoginWithWrongTokenIsForbidden(t *testing.T) {
	h := start(t)
	for _, q := range []string{"?token=nope", "?token=", "?token=" + strings.Repeat("0", 64)} {
		resp := get(t, h.base+"/"+q, false)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s: status %d", q, resp.StatusCode)
		}
		if len(resp.Cookies()) != 0 {
			t.Fatalf("%s: cookie set on wrong token", q)
		}
	}
}

func TestStaticRequiresCookie(t *testing.T) {
	h := start(t)
	for _, path := range []string{"/", "/app.js", "/manifest.webmanifest"} {
		resp := get(t, h.base+path, false)
		if resp.StatusCode != http.StatusForbidden {
			t.Fatalf("%s without cookie: status %d", path, resp.StatusCode)
		}
		body, _ := io.ReadAll(resp.Body)
		if !strings.Contains(string(body), "skye web") {
			t.Fatalf("%s: body %q does not point to skye web", path, body)
		}
	}
	resp := get(t, h.base+"/", true)
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), "skye") {
		t.Fatalf("index with cookie: status %d body %q", resp.StatusCode, body)
	}
	if resp := get(t, h.base+"/app.js", true); resp.StatusCode != http.StatusOK {
		t.Fatalf("app.js with cookie: status %d", resp.StatusCode)
	}
	bad, _ := http.NewRequest(http.MethodGet, h.base+"/app.js", nil)
	bad.AddCookie(&http.Cookie{Name: CookieName, Value: strings.Repeat("0", 64)})
	r2, err := noRedirect().Do(bad)
	if err != nil {
		t.Fatal(err)
	}
	r2.Body.Close()
	if r2.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong cookie: status %d", r2.StatusCode)
	}
}

func dial(t *testing.T, h harness, cookie bool, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	header := http.Header{}
	if cookie {
		header.Set("Cookie", CookieName+"="+testToken)
	}
	if origin != "" {
		header.Set("Origin", origin)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(h.base, "http")+"/ws", &websocket.DialOptions{HTTPHeader: header})
	if conn != nil {
		t.Cleanup(func() { conn.CloseNow() })
	}
	return conn, resp, err
}

func TestWebSocketRequiresCookie(t *testing.T) {
	h := start(t)
	_, resp, err := dial(t, h, false, fmt.Sprintf("http://localhost:%d", h.port))
	if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
		t.Fatalf("err=%v resp=%v", err, resp)
	}
}

func TestWebSocketRequiresExactOrigin(t *testing.T) {
	h := start(t)
	for _, origin := range []string{
		"",
		"http://evil.example",
		fmt.Sprintf("http://localhost:%d.evil.example", h.port),
		fmt.Sprintf("https://localhost:%d", h.port),
		fmt.Sprintf("http://localhost:%d", h.port+1),
		"http://localhost",
		"null",
	} {
		_, resp, err := dial(t, h, true, origin)
		if err == nil || resp == nil || resp.StatusCode != http.StatusForbidden {
			t.Fatalf("origin %q: err=%v resp=%v", origin, err, resp)
		}
	}
	for _, origin := range []string{fmt.Sprintf("http://localhost:%d", h.port), fmt.Sprintf("http://127.0.0.1:%d", h.port)} {
		if _, _, err := dial(t, h, true, origin); err != nil {
			t.Fatalf("origin %q: %v", origin, err)
		}
	}
}

func open(t *testing.T, h harness) *websocket.Conn {
	t.Helper()
	conn, _, err := dial(t, h, true, fmt.Sprintf("http://localhost:%d", h.port))
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func send(t *testing.T, conn *websocket.Conn, frame string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := conn.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
		t.Fatal(err)
	}
}

func recv(t *testing.T, conn *websocket.Conn) map[string]json.RawMessage {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("frame %q: %v", data, err)
	}
	return m
}

func TestRequestGetsReplyWithSameID(t *testing.T) {
	h := start(t)
	conn := open(t, h)
	send(t, conn, `{"id":42,"method":"Echo","args":["oi"]}`)
	m := recv(t, conn)
	if string(m["id"]) != "42" || string(m["result"]) != `"oi"` {
		t.Fatalf("reply = %v", m)
	}
	if _, ok := m["error"]; ok {
		t.Fatalf("unexpected error field: %v", m)
	}
}

func TestErrorsReplyAndKeepSocketUsable(t *testing.T) {
	h := start(t)
	conn := open(t, h)
	for i, frame := range []string{
		`{"id":1,"method":"Nope","args":[]}`,
		`{"id":2,"method":"Echo","args":[]}`,
		`{"id":3,"method":"Panic","args":[]}`,
	} {
		send(t, conn, frame)
		m := recv(t, conn)
		if string(m["id"]) != fmt.Sprint(i+1) || len(m["error"]) == 0 {
			t.Fatalf("reply to %s = %v", frame, m)
		}
		if _, ok := m["result"]; ok {
			t.Fatalf("result alongside error: %v", m)
		}
	}
	send(t, conn, `not json`)
	if m := recv(t, conn); string(m["id"]) != "null" || len(m["error"]) == 0 {
		t.Fatalf("reply to garbage = %v", m)
	}
	send(t, conn, `{"id":4,"method":"Echo","args":["ainda aqui"]}`)
	if m := recv(t, conn); string(m["id"]) != "4" || string(m["result"]) != `"ainda aqui"` {
		t.Fatalf("reply = %v", m)
	}
}

func TestBroadcastReachesEveryClient(t *testing.T) {
	h := start(t)
	a, b := open(t, h), open(t, h)
	deadline := time.Now().Add(5 * time.Second)
	for h.srv.Clients() < 2 {
		if time.Now().After(deadline) {
			t.Fatalf("clients = %d", h.srv.Clients())
		}
		time.Sleep(5 * time.Millisecond)
	}
	h.srv.Broadcast("terminals", []string{"t1"})
	for _, conn := range []*websocket.Conn{a, b} {
		m := recv(t, conn)
		if string(m["event"]) != `"terminals"` || string(m["data"]) != `["t1"]` {
			t.Fatalf("event = %v", m)
		}
	}
}

func TestClosedClientIsUnregistered(t *testing.T) {
	h := start(t)
	conn := open(t, h)
	deadline := time.Now().Add(5 * time.Second)
	for h.srv.Clients() < 1 {
		if time.Now().After(deadline) {
			t.Fatal("client never registered")
		}
		time.Sleep(5 * time.Millisecond)
	}
	conn.Close(websocket.StatusNormalClosure, "")
	for h.srv.Clients() != 0 {
		if time.Now().After(deadline) {
			t.Fatalf("clients = %d after close", h.srv.Clients())
		}
		time.Sleep(5 * time.Millisecond)
	}
}

func TestManifestIsServedAsManifest(t *testing.T) {
	h := start(t)
	resp := get(t, h.base+"/manifest.webmanifest", true)
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Content-Type") != "application/manifest+json" {
		t.Fatalf("status %d content-type %q", resp.StatusCode, resp.Header.Get("Content-Type"))
	}
}
