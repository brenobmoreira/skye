package winshell

import (
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
)

func fallback() fstest.MapFS {
	return fstest.MapFS{"index.html": {Data: []byte("<!doctype html><html lang=\"pt-BR\"><body>embutida</body></html>")}}
}

func get(t *testing.T, h http.Handler, path string) (int, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	body, _ := io.ReadAll(rec.Result().Body)
	return rec.Code, string(body)
}

func TestProxyServesWhatSkyeWebServesWithTheToken(t *testing.T) {
	var cookie string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("skye_token"); err == nil {
			cookie = c.Value
		}
		io.WriteString(w, "do servidor "+r.URL.Path)
	}))
	defer srv.Close()
	_, portText, _ := net.SplitHostPort(strings.TrimPrefix(srv.URL, "http://"))
	port, _ := strconv.Atoi(portText)
	reads := 0
	p := &Proxy{Port: port, Cookie: "skye_token", Token: func() (string, error) { reads++; return "t0k", nil }, Probe: Probe, Fallback: fallback()}
	for range 2 {
		if code, body := get(t, p, "/assets/index-abc.js"); code != 200 || body != "do servidor /assets/index-abc.js" {
			t.Fatalf("%d %q", code, body)
		}
	}
	if cookie != "t0k" || reads != 1 {
		t.Fatalf("cookie %q reads %d", cookie, reads)
	}
}

func TestProxyFallsBackToTheBuiltInCopyWhileSkyeWebIsDown(t *testing.T) {
	ln, _ := net.Listen("tcp4", "127.0.0.1:0")
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	p := &Proxy{Port: port, Cookie: "skye_token", Token: func() (string, error) { return "t0k", nil }, Probe: Probe, Fallback: fallback()}
	code, body := get(t, p, "/")
	if code != 200 || !strings.Contains(body, `<html data-fallback="1" lang="pt-BR">`) || !strings.Contains(body, "embutida") {
		t.Fatalf("%d %q", code, body)
	}
}
