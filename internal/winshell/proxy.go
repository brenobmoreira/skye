package winshell

import (
	"bytes"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"
	"time"
)

// Proxy is what skye.exe shows: the interface the skye web in WSL serves, so updating skye in
// WSL updates the window too. While that server is down it serves the copy built into the exe,
// marked with data-fallback so the page reloads once the server is up.
type Proxy struct {
	Port int
	// Cookie is the name the server reads the token from (web.CookieName).
	Cookie string
	// Token reads the server's token; it runs wsl.exe, so the result is kept.
	Token    func() (string, error)
	Probe    func(int, time.Duration) bool
	Fallback fs.FS

	mu    sync.Mutex
	token string
}

func (p *Proxy) cachedToken() (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.token != "" {
		return p.token, nil
	}
	t, err := p.Token()
	if err != nil {
		return "", err
	}
	p.token = t
	return t, nil
}

func (p *Proxy) forget() {
	p.mu.Lock()
	p.token = ""
	p.mu.Unlock()
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.Probe(p.Port, ProbeTimeout) {
		p.serveFallback(w, r)
		return
	}
	token, err := p.cachedToken()
	if err != nil {
		p.serveFallback(w, r)
		return
	}
	target := &url.URL{Scheme: "http", Host: fmt.Sprintf("127.0.0.1:%d", p.Port)}
	rp := &httputil.ReverseProxy{
		Rewrite: func(pr *httputil.ProxyRequest) {
			pr.SetURL(target)
			pr.Out.Host = target.Host
			pr.Out.Header.Del("Cookie")
			pr.Out.AddCookie(&http.Cookie{Name: p.Cookie, Value: token})
		},
		ModifyResponse: func(res *http.Response) error {
			if res.StatusCode == http.StatusForbidden {
				// The token changed under us: read it again next time.
				p.forget()
			}
			res.Header.Set("Cache-Control", "no-cache")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, _ error) { p.serveFallback(w, r) },
	}
	rp.ServeHTTP(w, r)
}

func (p *Proxy) serveFallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		page, err := fs.ReadFile(p.Fallback, "index.html")
		if err != nil {
			http.Error(w, "sem interface embutida", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(bytes.Replace(page, []byte("<html"), []byte(`<html data-fallback="1"`), 1))
		return
	}
	http.FileServer(http.FS(p.Fallback)).ServeHTTP(w, r)
}
