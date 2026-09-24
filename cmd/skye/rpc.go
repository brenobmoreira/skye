package main

import (
	"encoding/json"
	"fmt"

	"github.com/brenobmoreira/skye/internal/config"
	"github.com/brenobmoreira/skye/internal/resume"
	"github.com/brenobmoreira/skye/internal/terminals"
	"github.com/brenobmoreira/skye/internal/web"
)

type rpcTarget interface {
	List() []terminals.Terminal
	Conversations() []resume.Conversation
	Presets() []config.Preset
	NewTerminal(preset string) (terminals.Terminal, error)
	Resume(sessionID string) (terminals.Terminal, error)
	Write(id, data string) error
	Resize(id string, cols, rows int) error
	Snapshot(id string) (string, error)
	Rename(id, name string) error
	Reorder(ids []string) error
	Close(id string) error
	Forget(sessionID string) error
	Sound() bool
	SetSound(on bool)
	SetFocused(focused bool)
	Quit() error
	Problems() []string
}

func decodeArgs(method string, args []json.RawMessage, dst ...any) error {
	if len(args) != len(dst) {
		return fmt.Errorf("%s: esperava %d argumentos, recebeu %d", method, len(dst), len(args))
	}
	for i, d := range dst {
		if err := json.Unmarshal(args[i], d); err != nil {
			return fmt.Errorf("%s: argumento %d inválido: %v", method, i+1, err)
		}
	}
	return nil
}

func dispatcher(t rpcTarget) web.Dispatch {
	return func(method string, args []json.RawMessage) (any, error) {
		var (
			a, b       string
			cols, rows int
			flag       bool
			list       []string
		)
		decode := func(dst ...any) error { return decodeArgs(method, args, dst...) }
		switch method {
		case "List":
			if err := decode(); err != nil {
				return nil, err
			}
			return t.List(), nil
		case "Conversations":
			if err := decode(); err != nil {
				return nil, err
			}
			return t.Conversations(), nil
		case "Presets":
			if err := decode(); err != nil {
				return nil, err
			}
			return t.Presets(), nil
		case "Sound":
			if err := decode(); err != nil {
				return nil, err
			}
			return t.Sound(), nil
		case "Problems":
			if err := decode(); err != nil {
				return nil, err
			}
			return t.Problems(), nil
		case "Quit":
			if err := decode(); err != nil {
				return nil, err
			}
			return nil, t.Quit()
		case "NewTerminal":
			if err := decode(&a); err != nil {
				return nil, err
			}
			return t.NewTerminal(a)
		case "Resume":
			if err := decode(&a); err != nil {
				return nil, err
			}
			return t.Resume(a)
		case "Snapshot":
			if err := decode(&a); err != nil {
				return nil, err
			}
			return t.Snapshot(a)
		case "Close":
			if err := decode(&a); err != nil {
				return nil, err
			}
			return nil, t.Close(a)
		case "Forget":
			if err := decode(&a); err != nil {
				return nil, err
			}
			return nil, t.Forget(a)
		case "Write":
			if err := decode(&a, &b); err != nil {
				return nil, err
			}
			return nil, t.Write(a, b)
		case "Rename":
			if err := decode(&a, &b); err != nil {
				return nil, err
			}
			return nil, t.Rename(a, b)
		case "Reorder":
			if err := decode(&list); err != nil {
				return nil, err
			}
			return nil, t.Reorder(list)
		case "Resize":
			if err := decode(&a, &cols, &rows); err != nil {
				return nil, err
			}
			return nil, t.Resize(a, cols, rows)
		case "SetSound":
			if err := decode(&flag); err != nil {
				return nil, err
			}
			t.SetSound(flag)
			return nil, nil
		case "SetFocused":
			if err := decode(&flag); err != nil {
				return nil, err
			}
			t.SetFocused(flag)
			return nil, nil
		}
		return nil, fmt.Errorf("método desconhecido: %q", method)
	}
}
