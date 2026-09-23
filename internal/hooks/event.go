package hooks

import (
	"encoding/json"
	"errors"
	"fmt"
)

type Event struct {
	Terminal  string
	Name      string
	SessionID string
	Cwd       string
	Prompt    string
	Message   string
}

var ErrNoTerminal = errors.New("hook without terminal id")

type payload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	Cwd           string `json:"cwd"`
	Prompt        string `json:"prompt"`
	Message       string `json:"message"`
}

func Parse(terminal string, body []byte) (Event, error) {
	if terminal == "" {
		return Event{}, ErrNoTerminal
	}
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return Event{}, fmt.Errorf("hook payload: %w", err)
	}
	if p.HookEventName == "" {
		return Event{}, errors.New("hook payload without hook_event_name")
	}
	return Event{
		Terminal:  terminal,
		Name:      p.HookEventName,
		SessionID: p.SessionID,
		Cwd:       p.Cwd,
		Prompt:    p.Prompt,
		Message:   p.Message,
	}, nil
}
