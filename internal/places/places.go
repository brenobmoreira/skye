// Package places keeps the folders the + menu opens claude in, in a small JSON file the user
// can also edit by hand.
package places

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type Place struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

type Store struct {
	mu    sync.Mutex
	file  string
	home  string
	items []Place
}

// Open reads the file; a missing file is an empty list, a broken one an empty list and an error.
func Open(file, home string) (*Store, error) {
	s := &Store{file: file, home: home, items: []Place{}}
	data, err := os.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(data, &s.items); err != nil {
		s.items = []Place{}
		return s, fmt.Errorf("%s: %w", file, err)
	}
	return s, nil
}

// Expand turns a leading ~ into the home folder.
func (s *Store) Expand(path string) string {
	if path == "~" || strings.HasPrefix(path, "~/") {
		return filepath.Join(s.home, path[1:])
	}
	return path
}

func (s *Store) List() []Place {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Place{}, s.items...)
}

func (s *Store) Get(name string) (Place, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(name)
	if i < 0 {
		return Place{}, false
	}
	return s.items[i], true
}

func (s *Store) index(name string) int {
	for i, p := range s.items {
		if strings.EqualFold(p.Name, strings.TrimSpace(name)) {
			return i
		}
	}
	return -1
}

func (s *Store) Add(name, path string) error {
	name, path = strings.TrimSpace(name), strings.TrimSpace(path)
	if name == "" {
		return errors.New("sem nome")
	}
	if path == "" {
		return errors.New("sem path")
	}
	info, err := os.Stat(s.Expand(path))
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s não existe", path)
	}
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s não é uma pasta", path)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index(name) >= 0 {
		return fmt.Errorf("%q já existe", name)
	}
	s.items = append(s.items, Place{Name: name, Path: path})
	return s.save()
}

func (s *Store) Remove(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := s.index(name)
	if i < 0 {
		return fmt.Errorf("path desconhecido: %q", name)
	}
	s.items = append(s.items[:i:i], s.items[i+1:]...)
	return s.save()
}

func (s *Store) save() error {
	if err := os.MkdirAll(filepath.Dir(s.file), 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s.items, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.file + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.file)
}
