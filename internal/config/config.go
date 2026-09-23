package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Preset struct {
	Name    string `toml:"name" json:"name"`
	Command string `toml:"command" json:"command"`
}

type Config struct {
	Shell   string   `toml:"shell"`
	Sound   bool     `toml:"sound"`
	Presets []Preset `toml:"preset"`
}

func Default() Config {
	return Config{Sound: true}
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
	}
}
