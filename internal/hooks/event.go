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
	// Reason says why a SessionEnd happened and Source why a SessionStart did; both are
	// "clear" around a /clear.
	Reason string
	Source string
	// Tool is the tool a PermissionRequest asks for, with its command, file or url.
	Tool string
	// Trigger says whether a PreCompact is manual or auto; AgentID names a subagent.
	Trigger string
	AgentID string
}

var ErrNoTerminal = errors.New("hook without terminal id")

type payload struct {
	HookEventName string `json:"hook_event_name"`
	SessionID     string `json:"session_id"`
	Cwd           string `json:"cwd"`
	Prompt        string `json:"prompt"`
	Message       string `json:"message"`
	Reason        string `json:"reason"`
	Source        string `json:"source"`
	Trigger       string `json:"trigger"`
	AgentID       string `json:"agent_id"`
	ToolName      string `json:"tool_name"`
	ToolInput     struct {
		Command  string `json:"command"`
		FilePath string `json:"file_path"`
		URL      string `json:"url"`
	} `json:"tool_input"`
}

func (p payload) tool() string {
	for _, detail := range []string{p.ToolInput.Command, p.ToolInput.FilePath, p.ToolInput.URL} {
		if detail != "" && p.ToolName != "" {
			return p.ToolName + ": " + detail
		}
	}
	return p.ToolName
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
		Reason:    p.Reason,
		Source:    p.Source,
		Tool:      p.tool(),
		Trigger:   p.Trigger,
		AgentID:   p.AgentID,
	}, nil
}
