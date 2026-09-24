package terminals

import (
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/brenobmoreira/skye/internal/hooks"
	"github.com/brenobmoreira/skye/internal/resume"
)

type State string

const (
	Shell   State = "shell"
	Running State = "running"
	Waiting State = "waiting"
	Idle    State = "idle"
)

const TitleLimit = 80

var rank = map[State]int{Waiting: 0, Idle: 1, Running: 2, Shell: 3}

var validEvents = map[string]bool{
	"SessionStart":     true,
	"UserPromptSubmit": true,
	"PostToolUse":      true,
	"Notification":     true,
	"Stop":             true,
	"SessionEnd":       true,
}

type Terminal struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Preset    string    `json:"preset"`
	Cwd       string    `json:"cwd"`
	Window    string    `json:"-"`
	Pane      string    `json:"-"`
	State     State     `json:"state"`
	SessionID string    `json:"sessionId"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	// Order sorts terminals inside the same state group; 0 means not placed yet.
	Order int `json:"order"`
}

func (t Terminal) Conversation(now time.Time) (resume.Conversation, bool) {
	if t.SessionID == "" {
		return resume.Conversation{}, false
	}
	title := t.Title
	if title == "" {
		title = t.Name
	}
	return resume.Conversation{SessionID: t.SessionID, Cwd: t.Cwd, Title: title, Preset: t.Preset, EndedAt: now}, true
}

type Change struct {
	Terminal  Terminal
	Prev      State
	Attention bool
	Ended     *resume.Conversation
}

type Registry struct {
	mu    sync.Mutex
	items map[string]*Terminal
	now   func() time.Time
}

func NewRegistry(now func() time.Time) *Registry {
	return &Registry{items: map[string]*Terminal{}, now: now}
}

func (r *Registry) Add(t Terminal) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = r.now()
	}
	if t.State == "" {
		t.State = Shell
	}
	if t.Order == 0 {
		t.Order = r.edge(1)
	}
	r.items[t.ID] = &t
}

// edge returns the order just past the last terminal (dir 1) or before the first (dir -1).
func (r *Registry) edge(dir int) int {
	if len(r.items) == 0 {
		return 1
	}
	limit := 0
	first := true
	for _, t := range r.items {
		if first || t.Order*dir > limit*dir {
			limit, first = t.Order, false
		}
	}
	return limit + dir
}

// Reorder places the given terminals first, in that order, and the others after them as they
// were. The state groups still come first, so a terminal cannot leave its group.
func (r *Registry) Reorder(ids []string) []Terminal {
	r.mu.Lock()
	defer r.mu.Unlock()
	placed := map[string]bool{}
	var order []*Terminal
	for _, id := range ids {
		if t, ok := r.items[id]; ok && !placed[id] {
			placed[id] = true
			order = append(order, t)
		}
	}
	for _, t := range r.sorted() {
		if !placed[t.ID] {
			order = append(order, r.items[t.ID])
		}
	}
	out := make([]Terminal, 0, len(order))
	for i, t := range order {
		t.Order = i + 1
		out = append(out, *t)
	}
	return out
}

func (r *Registry) Get(id string) (Terminal, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[id]
	if !ok {
		return Terminal{}, false
	}
	return *t, true
}

func (r *Registry) find(match func(*Terminal) bool) (Terminal, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, t := range r.items {
		if match(t) {
			return *t, true
		}
	}
	return Terminal{}, false
}

func (r *Registry) ByPane(pane string) (Terminal, bool) {
	return r.find(func(t *Terminal) bool { return t.Pane == pane })
}

func (r *Registry) ByWindow(window string) (Terminal, bool) {
	return r.find(func(t *Terminal) bool { return t.Window == window })
}

func (r *Registry) Remove(id string) (Terminal, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[id]
	if !ok {
		return Terminal{}, false
	}
	delete(r.items, id)
	return *t, true
}

func (r *Registry) Rename(id, name string) (Terminal, bool) {
	name = strings.TrimSpace(name)
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[id]
	if !ok || name == "" {
		return Terminal{}, false
	}
	t.Name = name
	return *t, true
}

func (r *Registry) List() []Terminal {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sorted()
}

func (r *Registry) sorted() []Terminal {
	list := make([]Terminal, 0, len(r.items))
	for _, t := range r.items {
		list = append(list, *t)
	}
	sort.Slice(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if rank[a.State] != rank[b.State] {
			return rank[a.State] < rank[b.State]
		}
		if a.Order != b.Order {
			return a.Order < b.Order
		}
		if !a.CreatedAt.Equal(b.CreatedAt) {
			return a.CreatedAt.Before(b.CreatedAt)
		}
		return a.ID < b.ID
	})
	return list
}

func (r *Registry) Apply(ev hooks.Event) (Change, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	t, ok := r.items[ev.Terminal]
	if !ok {
		return Change{}, false
	}
	if !validEvents[ev.Name] {
		return Change{}, false
	}
	prev := t.State
	ch := Change{Prev: prev}
	now := r.now()
	switch ev.Name {
	case "SessionStart":
		if t.SessionID != "" && ev.SessionID != "" && t.SessionID != ev.SessionID {
			// After /clear the terminal keeps going as the same session for the user.
			if c, ok := t.Conversation(now); ok && ev.Source != "clear" {
				ch.Ended = &c
			}
			t.Title = ""
		}
		if ev.Cwd != "" {
			t.Cwd = ev.Cwd
		}
		t.State = Idle
	case "UserPromptSubmit":
		if t.Title == "" {
			t.Title = titleFrom(ev.Prompt)
		}
		if ev.Cwd != "" {
			t.Cwd = ev.Cwd
		}
		t.State = Running
	case "PostToolUse":
		if ev.Cwd != "" {
			t.Cwd = ev.Cwd
		}
		t.State = Running
	case "Notification":
		if ev.Cwd != "" {
			t.Cwd = ev.Cwd
		}
		if prev == Running {
			t.State = Waiting
			ch.Attention = true
		}
	case "Stop":
		if ev.Cwd != "" {
			t.Cwd = ev.Cwd
		}
		t.State = Idle
		ch.Attention = prev != Idle
	case "SessionEnd":
		if ev.Cwd != "" {
			t.Cwd = ev.Cwd
		}
		if ev.Reason == "clear" {
			t.State = Idle
			ch.Terminal = *t
			return ch, true
		}
		if c, ok := t.Conversation(now); ok {
			ch.Ended = &c
		}
		t.SessionID, t.Title, t.State = "", "", Shell
		ch.Terminal = *t
		return ch, true
	}
	if ev.SessionID != "" {
		t.SessionID = ev.SessionID
	}
	if prev == Running && (t.State == Waiting || t.State == Idle) {
		t.Order = r.edge(-1)
	}
	ch.Terminal = *t
	return ch, true
}

func titleFrom(prompt string) string {
	for _, line := range strings.Split(prompt, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		runes := []rune(line)
		if len(runes) > TitleLimit {
			return string(runes[:TitleLimit-1]) + "…"
		}
		return line
	}
	return ""
}
