package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Preset struct {
	Name    string `toml:"name" json:"name"`
	Command string `toml:"command" json:"command"`
}

// Repo is a repository the + menu can open a new worktree of.
type Repo struct {
	Name    string `toml:"name" json:"name"`
	Path    string `toml:"path" json:"-"`
	Base    string `toml:"base" json:"-"`
	Dir     string `toml:"dir" json:"-"`
	Command string `toml:"command" json:"-"`
}

// Resolved fills the defaults and expands a leading ~: the base is the repo's HEAD, worktrees go
// next to the repo and run claude.
func (r Repo) Resolved(home string) Repo {
	expand := func(p string) string {
		if p == "~" || strings.HasPrefix(p, "~/") {
			return filepath.Join(home, p[1:])
		}
		return p
	}
	r.Path = expand(r.Path)
	r.Dir = expand(r.Dir)
	if r.Dir == "" {
		r.Dir = filepath.Dir(r.Path)
	}
	if r.Base == "" {
		r.Base = "HEAD"
	}
	if r.Command == "" {
		r.Command = "claude"
	}
	return r
}

type Config struct {
	Shell   string   `toml:"shell"`
	Sound   bool     `toml:"sound"`
	Presets []Preset `toml:"preset"`
	Repos   []Repo   `toml:"repo"`
	WebPort int      `toml:"web_port"`
}

const DefaultWebPort = 7810

func Default() Config {
	return Config{Sound: true, WebPort: DefaultWebPort}
}

func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Default(), err
	}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return Default(), fmt.Errorf("%s: %w", path, err)
	}
	if err := cfg.validate(); err != nil {
		return Default(), fmt.Errorf("%s: %w", path, err)
	}
	return cfg, nil
}

func (c Config) validate() error {
	if c.WebPort < 1024 || c.WebPort > 65535 {
		return fmt.Errorf("web_port %d is outside 1024-65535", c.WebPort)
	}
	seen := map[string]bool{}
	for i, p := range c.Presets {
		if p.Name == "" {
			return fmt.Errorf("preset #%d has no name", i+1)
		}
		if seen[p.Name] {
			return fmt.Errorf("preset %q is duplicated", p.Name)
		}
		seen[p.Name] = true
	}
	repos := map[string]bool{}
	for i, r := range c.Repos {
		if r.Name == "" || r.Path == "" {
			return fmt.Errorf("repo #%d needs name and path", i+1)
		}
		if repos[r.Name] {
			return fmt.Errorf("repo %q is duplicated", r.Name)
		}
		repos[r.Name] = true
	}
	return nil
}

func (c Config) Preset(name string) (Preset, bool) {
	for _, p := range c.Presets {
		if p.Name == name {
			return p, true
		}
	}
	return Preset{}, false
}

func (c Config) Repo(name string) (Repo, bool) {
	for _, r := range c.Repos {
		if r.Name == name {
			return r, true
		}
	}
	return Repo{}, false
}

func (c Config) ResolveShell(getenv func(string) string) string {
	if c.Shell != "" {
		return c.Shell
	}
	if s := getenv("SHELL"); s != "" {
		return s
	}
	return "/bin/bash"
}

type Paths struct {
	ConfigFile    string
	TmuxConf      string
	StateDir      string
	LaunchDir     string
	Conversations string
	Socket        string
	WebToken      string
}

func DefaultPaths(getenv func(string) string) Paths {
	home := getenv("HOME")
	configHome := getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	stateHome := getenv("XDG_STATE_HOME")
	if stateHome == "" {
		stateHome = filepath.Join(home, ".local", "state")
	}
	configDir := filepath.Join(configHome, "skye")
	stateDir := filepath.Join(stateHome, "skye")
	runtimeDir := getenv("XDG_RUNTIME_DIR")
	if runtimeDir == "" {
		runtimeDir = stateDir
	}
	return Paths{
		ConfigFile:    filepath.Join(configDir, "config.toml"),
		TmuxConf:      filepath.Join(configDir, "tmux.conf"),
		StateDir:      stateDir,
		LaunchDir:     filepath.Join(stateDir, "launch"),
		Conversations: filepath.Join(stateDir, "conversations.json"),
		Socket:        filepath.Join(runtimeDir, "skye.sock"),
		WebToken:      filepath.Join(configDir, "web-token"),
	}
}
