package resume

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"time"
)

const Limit = 50

type Conversation struct {
	SessionID string    `json:"sessionId"`
	Cwd       string    `json:"cwd"`
	Title     string    `json:"title"`
	Preset    string    `json:"preset"`
	EndedAt   time.Time `json:"endedAt"`
}

var sessionIDPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)

func ValidSessionID(id string) bool {
	return sessionIDPattern.MatchString(id)
}

func Command(id string) (string, error) {
	if !ValidSessionID(id) {
		return "", fmt.Errorf("invalid session id %q", id)
	}
	return "claude --resume " + id, nil
}

type Store struct {
	mu    sync.Mutex
	path  string
	items []Conversation
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s.items); err != nil {
		s.items = nil
		return s, fmt.Errorf("%s: %w", path, err)
	}
	return s, nil
}

func (s *Store) Add(c Conversation) error {
	if !ValidSessionID(c.SessionID) {
		return fmt.Errorf("invalid session id %q", c.SessionID)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	items := []Conversation{c}
	for _, it := range s.items {
		if it.SessionID != c.SessionID {
			items = append(items, it)
		}
	}
	if len(items) > Limit {
		items = items[:Limit]
	}
	s.items = items
	return s.save()
}

func (s *Store) Remove(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	items := s.items[:0:0]
	for _, it := range s.items {
		if it.SessionID != id {
			items = append(items, it)
		}
	}
	s.items = items
	return s.save()
}

func (s *Store) Get(id string) (Conversation, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, it := range s.items {
		if it.SessionID == id {
			return it, true
		}
	}
	return Conversation{}, false
}

func (s *Store) List() []Conversation {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Conversation{}, s.items...)
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
