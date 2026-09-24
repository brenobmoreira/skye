package config

import (
	"os"
	"path/filepath"
	"testing"
)

func write(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.Sound || cfg.Shell != "" || len(cfg.Presets) != 0 {
		t.Fatalf("got %+v, want defaults", cfg)
	}
}

func TestLoadPresets(t *testing.T) {
	path := write(t, `
shell = "/bin/zsh"

[[preset]]
name = "blog"
command = "cd ~/projects/blog && claude"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Shell != "/bin/zsh" || !cfg.Sound {
		t.Fatalf("got %+v", cfg)
	}
	p, ok := cfg.Preset("blog")
	if !ok || p.Command != "cd ~/projects/blog && claude" {
		t.Fatalf("preset = %+v, %v", p, ok)
	}
	if _, ok := cfg.Preset("other"); ok {
		t.Fatal("unknown preset found")
	}
}

func TestLoadSoundOff(t *testing.T) {
	cfg, err := Load(write(t, "sound = false\n"))
	if err != nil || cfg.Sound {
		t.Fatalf("cfg=%+v err=%v", cfg, err)
	}
}

func TestLoadInvalidTomlFallsBackToDefaults(t *testing.T) {
	cfg, err := Load(write(t, "shell = \n"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !cfg.Sound || cfg.Shell != "" {
		t.Fatalf("got %+v, want defaults", cfg)
	}
}

func TestLoadRejectsDuplicateAndNamelessPresets(t *testing.T) {
	for _, body := range []string{
		"[[preset]]\nname = \"a\"\n[[preset]]\nname = \"a\"\n",
		"[[preset]]\ncommand = \"claude\"\n",
	} {
		if _, err := Load(write(t, body)); err == nil {
			t.Fatalf("expected error for %q", body)
		}
	}
}

func TestResolveShell(t *testing.T) {
	env := func(v map[string]string) func(string) string {
		return func(k string) string { return v[k] }
	}
	if got := (Config{Shell: "/bin/fish"}).ResolveShell(env(nil)); got != "/bin/fish" {
		t.Fatalf("got %q", got)
	}
	if got := (Config{}).ResolveShell(env(map[string]string{"SHELL": "/bin/zsh"})); got != "/bin/zsh" {
		t.Fatalf("got %q", got)
	}
	if got := (Config{}).ResolveShell(env(nil)); got != "/bin/bash" {
		t.Fatalf("got %q", got)
	}
}

func TestDefaultPaths(t *testing.T) {
	p := DefaultPaths(func(k string) string {
		return map[string]string{"HOME": "/home/demo", "XDG_RUNTIME_DIR": "/run/user/1000"}[k]
	})
	want := Paths{
		ConfigFile:    "/home/demo/.config/skye/config.toml",
		TmuxConf:      "/home/demo/.config/skye/tmux.conf",
		StateDir:      "/home/demo/.local/state/skye",
		LaunchDir:     "/home/demo/.local/state/skye/launch",
		Conversations: "/home/demo/.local/state/skye/conversations.json",
		Socket:        "/run/user/1000/skye.sock",
		WebToken:      "/home/demo/.config/skye/web-token",
	}
	if p != want {
		t.Fatalf("got %+v\nwant %+v", p, want)
	}
}

func TestDefaultPathsHonorsXDGAndMissingRuntimeDir(t *testing.T) {
	p := DefaultPaths(func(k string) string {
		return map[string]string{
			"HOME":            "/home/demo",
			"XDG_CONFIG_HOME": "/cfg",
			"XDG_STATE_HOME":  "/state",
		}[k]
	})
	if p.WebToken != "/cfg/skye/web-token" {
		t.Fatalf("web token = %q", p.WebToken)
	}
	if p.ConfigFile != "/cfg/skye/config.toml" || p.StateDir != "/state/skye" || p.Socket != "/state/skye/skye.sock" {
		t.Fatalf("got %+v", p)
	}
}

func TestLoadWebPortDefaultAndOverride(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil || cfg.WebPort != 7810 {
		t.Fatalf("default web port = %d, err=%v", cfg.WebPort, err)
	}
	cfg, err = Load(write(t, "web_port = 9000\n"))
	if err != nil || cfg.WebPort != 9000 {
		t.Fatalf("web port = %d, err=%v", cfg.WebPort, err)
	}
}

func TestLoadRejectsWebPortOutOfRange(t *testing.T) {
	for _, body := range []string{"web_port = 80\n", "web_port = 70000\n", "web_port = 1023\n"} {
		cfg, err := Load(write(t, body))
		if err == nil {
			t.Fatalf("expected error for %q", body)
		}
		if cfg.WebPort != 7810 {
			t.Fatalf("fallback web port = %d", cfg.WebPort)
		}
	}
	for _, body := range []string{"web_port = 1024\n", "web_port = 65535\n"} {
		if _, err := Load(write(t, body)); err != nil {
			t.Fatalf("unexpected error for %q: %v", body, err)
		}
	}
}

func TestLoadRepos(t *testing.T) {
	cfg, err := Load(write(t, `
[[repo]]
name = "livia"
path = "~/projects/ai_livia_copilot"
base = "origin/develop"

[[repo]]
name = "skye"
path = "/home/demo/skye"
dir = "~/wt"
command = "claude --model opus"
`))
	if err != nil {
		t.Fatal(err)
	}
	livia, ok := cfg.Repo("livia")
	if !ok || livia.Path != "~/projects/ai_livia_copilot" || livia.Base != "origin/develop" {
		t.Fatalf("livia = %+v %v", livia, ok)
	}
	if _, ok := cfg.Repo("nope"); ok {
		t.Fatal("unknown repo found")
	}
	got := livia.Resolved("/home/demo")
	if got.Path != "/home/demo/projects/ai_livia_copilot" || got.Dir != "/home/demo/projects" || got.Command != "claude" {
		t.Fatalf("resolved livia = %+v", got)
	}
	skye, _ := cfg.Repo("skye")
	got = skye.Resolved("/home/demo")
	if got.Base != "HEAD" || got.Dir != "/home/demo/wt" || got.Command != "claude --model opus" {
		t.Fatalf("resolved skye = %+v", got)
	}
}

func TestLoadRejectsBadRepos(t *testing.T) {
	for _, body := range []string{
		"[[repo]]\npath = \"/x\"\n",
		"[[repo]]\nname = \"a\"\n",
		"[[repo]]\nname = \"a\"\npath = \"/x\"\n[[repo]]\nname = \"a\"\npath = \"/y\"\n",
	} {
		if _, err := Load(write(t, body)); err == nil {
			t.Errorf("accepted %q", body)
		}
	}
}
