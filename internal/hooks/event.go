package hooks

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Event struct {
	Terminal  string
	Provider  string
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
	Usage   *Usage
	Context *Session
}

var ErrNoTerminal = errors.New("hook without terminal id")

type payload struct {
	HookEventName  string   `json:"hook_event_name"`
	SessionID      string   `json:"session_id"`
	ConversationID string   `json:"conversationId"`
	Cwd            string   `json:"cwd"`
	WorkspacePaths []string `json:"workspacePaths"`
	Prompt         string   `json:"prompt"`
	LastUserInput  string   `json:"lastUserInput"`
	Message        string   `json:"message"`
	Reason         string   `json:"reason"`
	Source         string   `json:"source"`
	Trigger        string   `json:"trigger"`
	AgentID        string   `json:"agent_id"`
	Model          string   `json:"model"`
	ModelName      string   `json:"modelName"`
	TranscriptPath string   `json:"transcript_path"`
	ToolName       string   `json:"tool_name"`
	ToolInput      struct {
		Command  string `json:"command"`
		FilePath string `json:"file_path"`
		URL      string `json:"url"`
	} `json:"tool_input"`
	ToolCall struct {
		Name string `json:"name"`
		Args struct {
			CommandLine string `json:"CommandLine"`
			Path        string `json:"Path"`
			URL         string `json:"URL"`
		} `json:"args"`
	} `json:"toolCall"`
}

func (p payload) tool() string {
	name := p.ToolName
	if name == "" {
		name = p.ToolCall.Name
	}
	for _, detail := range []string{
		p.ToolInput.Command, p.ToolInput.FilePath, p.ToolInput.URL,
		p.ToolCall.Args.CommandLine, p.ToolCall.Args.Path, p.ToolCall.Args.URL,
	} {
		if detail != "" && name != "" {
			return name + ": " + detail
		}
	}
	return name
}

func Parse(terminal string, body []byte) (Event, error) {
	return ParseEvent(terminal, "", body)
}

func ParseEvent(terminal, eventHint string, body []byte) (Event, error) {
	if terminal == "" {
		return Event{}, ErrNoTerminal
	}
	var p payload
	if err := json.Unmarshal(body, &p); err != nil {
		return Event{}, fmt.Errorf("hook payload: %w", err)
	}
	name := p.HookEventName
	if name == "" {
		name = strings.TrimSpace(eventHint)
	}
	if name == "PreInvocation" {
		name = "UserPromptSubmit"
	}
	if name == "" {
		return Event{}, errors.New("hook payload without hook_event_name")
	}
	sessionID := p.SessionID
	if sessionID == "" {
		sessionID = p.ConversationID
	}
	cwd := p.Cwd
	if cwd == "" && len(p.WorkspacePaths) > 0 {
		cwd = p.WorkspacePaths[0]
	}
	prompt := p.Prompt
	if prompt == "" {
		prompt = p.LastUserInput
	}
	var provider string
	if eventHint != "" || p.ConversationID != "" || len(p.WorkspacePaths) > 0 || p.LastUserInput != "" || p.ModelName != "" {
		provider = "agy"
	} else if p.Model != "" {
		provider = "codex"
	}
	var ctx *Session
	model := p.ModelName
	if model == "" {
		model = p.Model
	}
	if model != "" {
		ctx = &Session{Model: model}
	}
	var usage *Usage
	if p.TranscriptPath != "" {
		if u, pct := readCodexTranscript(p.TranscriptPath); u != nil || pct != nil {
			if provider == "" {
				provider = "codex"
			}
			usage = u
			if pct != nil {
				if ctx == nil {
					ctx = &Session{}
				}
				ctx.ContextPct = pct
			}
		}
	}
	return Event{
		Terminal:  terminal,
		Provider:  provider,
		Name:      name,
		SessionID: sessionID,
		Cwd:       cwd,
		Prompt:    prompt,
		Message:   p.Message,
		Reason:    p.Reason,
		Source:    p.Source,
		Tool:      p.tool(),
		Trigger:   p.Trigger,
		AgentID:   p.AgentID,
		Usage:     usage,
		Context:   ctx,
	}, nil
}

type codexRolloutLine struct {
	Type    string `json:"type"`
	Payload struct {
		Type string `json:"type"`
		Info struct {
			TotalTokenUsage struct {
				TotalTokens float64 `json:"total_tokens"`
			} `json:"total_token_usage"`
			LastTokenUsage struct {
				TotalTokens float64 `json:"total_tokens"`
			} `json:"last_token_usage"`
			ModelContextWindow float64 `json:"model_context_window"`
		} `json:"info"`
		RateLimits struct {
			Primary   *rawWindow `json:"primary"`
			Secondary *rawWindow `json:"secondary"`
		} `json:"rate_limits"`
	} `json:"payload"`
}

func readCodexTranscript(path string) (*Usage, *float64) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var u *Usage
	var pct *float64
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		if !bytes.Contains(line, []byte(`"token_count"`)) {
			continue
		}
		var row codexRolloutLine
		if json.Unmarshal(line, &row) != nil || row.Payload.Type != "token_count" {
			continue
		}
		fh := row.Payload.RateLimits.Primary.window()
		sd := row.Payload.RateLimits.Secondary.window()
		if fh != nil || sd != nil {
			u = &Usage{FiveHour: fh, SevenDay: sd}
		}
		if win := row.Payload.Info.ModelContextWindow; win > 0 {
			tokens := row.Payload.Info.LastTokenUsage.TotalTokens
			if tokens <= 0 {
				tokens = row.Payload.Info.TotalTokenUsage.TotalTokens
			}
			if tokens > 0 {
				v := (tokens / win) * 100
				pct = &v
			}
		}
	}
	return u, pct
}
