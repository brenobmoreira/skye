package hooks

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var Events = []string{"SessionStart", "UserPromptSubmit", "PostToolUse", "Notification", "Stop", "SessionEnd"}

const marker = "http://skye/event"

func Command() string {
	return `[ -n "$SKYE_TERMINAL_ID" ] && curl -s -m 2 --unix-socket "$SKYE_SOCKET" -X POST "` + marker +
		`?t=$SKYE_TERMINAL_ID" -H 'Content-Type: application/json' -d @- || true`
}

func Install(path string) (string, error) {
	settings := map[string]any{}
	data, err := os.ReadFile(path)
	existed := err == nil
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	if existed && len(bytes.TrimSpace(data)) > 0 {
		if err := json.Unmarshal(data, &settings); err != nil {
			return "", fmt.Errorf("%s não é JSON válido: %w", path, err)
		}
	}
	hooksMap := map[string]any{}
	if raw, ok := settings["hooks"]; ok && raw != nil {
		m, ok := raw.(map[string]any)
		if !ok {
			return "", fmt.Errorf("%s: \"hooks\" não é um objeto", path)
		}
		hooksMap = m
	}
	for _, ev := range Events {
		groups, err := upsert(hooksMap[ev])
		if err != nil {
			return "", fmt.Errorf("%s: hooks.%s: %w", path, ev, err)
		}
		hooksMap[ev] = groups
	}
	settings["hooks"] = hooksMap
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return "", err
	}
	backup := ""
	if existed {
		backup = path + ".bak"
		if err := os.WriteFile(backup, data, 0o600); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(out, '\n'), 0o600); err != nil {
		return "", err
	}
	return backup, os.Rename(tmp, path)
}

func upsert(v any) ([]any, error) {
	var groups []any
	if v != nil {
		g, ok := v.([]any)
		if !ok {
			return nil, errors.New("não é uma lista")
		}
		groups = g
	}
	cmd := map[string]any{"type": "command", "command": Command()}
	for _, g := range groups {
		gm, ok := g.(map[string]any)
		if !ok {
			continue
		}
		hs, _ := gm["hooks"].([]any)
		for i, h := range hs {
			hm, ok := h.(map[string]any)
			if !ok {
				continue
			}
			if c, _ := hm["command"].(string); strings.Contains(c, marker) {
				hs[i] = cmd
				gm["hooks"] = hs
				return groups, nil
			}
		}
	}
	return append(groups, map[string]any{"hooks": []any{cmd}}), nil
}
