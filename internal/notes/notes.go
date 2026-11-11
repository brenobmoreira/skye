// Package notes keeps the focus panel's list in a JSON file the user can also edit by hand. Every
// change reads the file again before applying, so a hand edit made while skye runs survives.
package notes

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/brenobmoreira/skye/internal/atomicfile"
)

type Item struct {
	ID     string  `json:"id"`
	Text   string  `json:"text"`
	Date   string  `json:"date,omitempty"`
	Time   string  `json:"time,omitempty"`
	Repeat *Repeat `json:"repeat,omitempty"`
	// Pinned notes sit in their own group at the top of the list.
	Pinned  bool      `json:"pinned,omitempty"`
	Created time.Time `json:"created"`
	// DoneAt is set only on notes that do not repeat; a repeating one leaves a Done instead.
	DoneAt time.Time `json:"doneAt,omitzero"`
}

// Done is one check of a repeating note. PrevDate is the date the note had before it moved on,
// so unchecking can put it back.
type Done struct {
	ID       string    `json:"id"`
	ItemID   string    `json:"itemId"`
	Text     string    `json:"text"`
	DoneAt   time.Time `json:"doneAt"`
	PrevDate string    `json:"prevDate"`
}

type Notes struct {
	Items []Item `json:"items"`
	Done  []Done `json:"done"`
}

type Edit struct {
	Text   string  `json:"text"`
	Date   string  `json:"date"`
	Time   string  `json:"time"`
	Repeat *Repeat `json:"repeat"`
}

var timePattern = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

type Store struct {
	mu       sync.Mutex
	file     string
	now      func() time.Time
	newID    func() string
	problems []string
}

// Open checks the file once. A missing file is an empty list; a broken one is moved aside, the
// list starts empty and the error says where the old file went.
func Open(file string, now func() time.Time, newID func() string) (*Store, error) {
	s := &Store{file: file, now: now, newID: newID}
	_, aside, err := s.load()
	if err != nil {
		return s, err
	}
	if aside != "" {
		return s, fmt.Errorf("%s ilegível, movido para %s", file, aside)
	}
	return s, nil
}

// load reads the file. A file that does not decode is renamed to <file>.broken-<time> and the
// list comes back empty; aside says where it went.
func (s *Store) load() (n Notes, aside string, err error) {
	n = Notes{Items: []Item{}, Done: []Done{}}
	data, err := os.ReadFile(s.file)
	if errors.Is(err, fs.ErrNotExist) {
		return n, "", nil
	}
	if err != nil {
		return n, "", err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	// A misspelled key would otherwise decode as an empty list and be written over.
	dec.DisallowUnknownFields()
	if err := dec.Decode(&n); err != nil {
		aside = s.file + ".broken-" + s.now().Format("20060102-150405")
		if err := os.Rename(s.file, aside); err != nil {
			return n, "", fmt.Errorf("%s ilegível e não consegui movê-lo: %w", s.file, err)
		}
		return Notes{Items: []Item{}, Done: []Done{}}, aside, nil
	}
	if n.Items == nil {
		n.Items = []Item{}
	}
	if n.Done == nil {
		n.Done = []Done{}
	}
	// A note added by hand without an id gets one now, saved, so it stays the same across reads.
	missing := false
	for i := range n.Items {
		if n.Items[i].ID == "" {
			n.Items[i].ID, missing = s.newID(), true
		}
	}
	if missing {
		if err := s.write(n); err != nil {
			return n, "", err
		}
	}
	return n, "", nil
}

func (s *Store) write(n Notes) error {
	data, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}
	return atomicfile.Write(s.file, data, 0o600)
}

func (s *Store) read() (Notes, error) {
	n, aside, err := s.load()
	if aside != "" {
		s.problems = append(s.problems, fmt.Sprintf("anotações ilegíveis, movidas para %s", aside))
	}
	return n, err
}

// change reads the file, applies f and writes the result; f's error leaves the file alone.
func (s *Store) change(f func(n *Notes) error) (Notes, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n, err := s.read()
	if err != nil {
		return n, err
	}
	if err := f(&n); err != nil {
		return n, err
	}
	return n, s.write(n)
}

func (s *Store) List() (Notes, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.read()
}

// Problems returns the files moved aside since the last call, and forgets them.
func (s *Store) Problems() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	p := s.problems
	s.problems = nil
	return p
}

func (s *Store) today() string { return s.now().Format(dateLayout) }

func findItem(n *Notes, id string) int {
	for i, it := range n.Items {
		if it.ID == id {
			return i
		}
	}
	return -1
}

func findDone(n *Notes, id string) int {
	for i, d := range n.Done {
		if d.ID == id {
			return i
		}
	}
	return -1
}

func (s *Store) Add(text string) (Item, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Item{}, errors.New("anotação vazia")
	}
	it := Item{ID: s.newID(), Text: text, Created: s.now()}
	_, err := s.change(func(n *Notes) error {
		n.Items = append(n.Items, it)
		return nil
	})
	return it, err
}

func validate(e Edit) (Edit, error) {
	e.Text = strings.TrimSpace(e.Text)
	if e.Text == "" {
		return e, errors.New("anotação vazia")
	}
	if e.Date != "" {
		if _, err := time.Parse(dateLayout, e.Date); err != nil {
			return e, fmt.Errorf("data inválida: %q", e.Date)
		}
	}
	if e.Time != "" {
		if e.Date == "" {
			return e, errors.New("hora sem data")
		}
		if !timePattern.MatchString(e.Time) {
			return e, fmt.Errorf("hora inválida: %q", e.Time)
		}
	}
	if e.Repeat != nil {
		if e.Date == "" {
			return e, errors.New("anotação que repete precisa de data")
		}
		if err := e.Repeat.Validate(); err != nil {
			return e, err
		}
	}
	return e, nil
}

func (s *Store) Edit(id string, e Edit) (Item, error) {
	e, err := validate(e)
	if err != nil {
		return Item{}, err
	}
	var out Item
	_, err = s.change(func(n *Notes) error {
		i := findItem(n, id)
		if i < 0 {
			return fmt.Errorf("anotação desconhecida: %q", id)
		}
		it := &n.Items[i]
		it.Text, it.Date, it.Time, it.Repeat = e.Text, e.Date, e.Time, e.Repeat
		if it.Repeat != nil {
			it.DoneAt = time.Time{}
		}
		out = *it
		return nil
	})
	return out, err
}

// Check marks a note done. A repeating one leaves a Done and moves to its next date after the
// later of today and its own date, so checking it early does not bring it back on the same day.
func (s *Store) Check(id string) error {
	_, err := s.change(func(n *Notes) error {
		i := findItem(n, id)
		if i < 0 {
			return fmt.Errorf("anotação desconhecida: %q", id)
		}
		it := &n.Items[i]
		if it.Repeat == nil {
			it.DoneAt = s.now()
			return nil
		}
		base := max(s.today(), it.Date)
		next, err := Next(*it.Repeat, base)
		if err != nil {
			return err
		}
		n.Done = append(n.Done, Done{ID: s.newID(), ItemID: it.ID, Text: it.Text, DoneAt: s.now(), PrevDate: it.Date})
		it.Date = next
		return nil
	})
	return err
}

var errNotToday = errors.New("só dá para desmarcar o que foi feito hoje")

// Uncheck undoes a check made today: on a plain note by its id, on a repeating one by the Done's id.
func (s *Store) Uncheck(id string) error {
	_, err := s.change(func(n *Notes) error {
		today := s.today()
		if i := findItem(n, id); i >= 0 {
			it := &n.Items[i]
			if it.DoneAt.IsZero() {
				return nil
			}
			if it.DoneAt.In(s.now().Location()).Format(dateLayout) != today {
				return errNotToday
			}
			it.DoneAt = time.Time{}
			return nil
		}
		j := findDone(n, id)
		if j < 0 {
			return fmt.Errorf("anotação desconhecida: %q", id)
		}
		d := n.Done[j]
		if d.DoneAt.In(s.now().Location()).Format(dateLayout) != today {
			return errNotToday
		}
		if i := findItem(n, d.ItemID); i >= 0 {
			n.Items[i].Date = d.PrevDate
		}
		n.Done = append(n.Done[:j:j], n.Done[j+1:]...)
		return nil
	})
	return err
}

func (s *Store) Delete(id string) error {
	_, err := s.change(func(n *Notes) error {
		if i := findItem(n, id); i >= 0 {
			n.Items = append(n.Items[:i:i], n.Items[i+1:]...)
			return nil
		}
		if j := findDone(n, id); j >= 0 {
			n.Done = append(n.Done[:j:j], n.Done[j+1:]...)
			return nil
		}
		return fmt.Errorf("anotação desconhecida: %q", id)
	})
	return err
}

func (s *Store) Pin(id string, pinned bool) error {
	_, err := s.change(func(n *Notes) error {
		i := findItem(n, id)
		if i < 0 {
			return fmt.Errorf("anotação desconhecida: %q", id)
		}
		n.Items[i].Pinned = pinned
		return nil
	})
	return err
}

// Move puts note id where note to is, shifting the ones between: the list's order is the order
// the panel shows inside each group.
func (s *Store) Move(id, to string) error {
	_, err := s.change(func(n *Notes) error {
		src, dst := findItem(n, id), findItem(n, to)
		if src < 0 || dst < 0 {
			return fmt.Errorf("anotação desconhecida: %q", map[bool]string{true: id, false: to}[src < 0])
		}
		moved := n.Items[src]
		n.Items = append(n.Items[:src:src], n.Items[src+1:]...)
		n.Items = append(n.Items[:dst:dst], append([]Item{moved}, n.Items[dst:]...)...)
		return nil
	})
	return err
}
