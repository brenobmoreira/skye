package notes

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fixture struct {
	file string
	now  time.Time
	ids  int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	return &fixture{file: filepath.Join(t.TempDir(), "notes.json"), now: time.Date(2026, 11, 11, 10, 0, 0, 0, time.Local)}
}

func (f *fixture) open(t *testing.T) *Store {
	t.Helper()
	s, err := Open(f.file, func() time.Time { return f.now }, func() string { f.ids++; return fmt.Sprintf("n%d", f.ids) })
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func (f *fixture) onDisk(t *testing.T) Notes {
	t.Helper()
	data, err := os.ReadFile(f.file)
	if err != nil {
		t.Fatal(err)
	}
	var n Notes
	if err := json.Unmarshal(data, &n); err != nil {
		t.Fatal(err)
	}
	return n
}

func mustList(t *testing.T, s *Store) Notes {
	t.Helper()
	n, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func TestOpenMissingFileIsEmpty(t *testing.T) {
	f := newFixture(t)
	n := mustList(t, f.open(t))
	if n.Items == nil || n.Done == nil || len(n.Items) != 0 || len(n.Done) != 0 {
		t.Fatalf("got %+v", n)
	}
}

func TestAddRejectsBlankText(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	if _, err := s.Add("   "); err == nil {
		t.Fatal("blank accepted")
	}
	if _, err := os.Stat(f.file); !os.IsNotExist(err) {
		t.Fatal("file written for a blank note")
	}
	it, err := s.Add("  ligar  ")
	if err != nil {
		t.Fatal(err)
	}
	if it.Text != "ligar" || it.ID != "n1" || !it.Created.Equal(f.now) {
		t.Fatalf("got %+v", it)
	}
	if got := f.onDisk(t).Items; len(got) != 1 || got[0].Text != "ligar" {
		t.Fatalf("on disk %+v", got)
	}
}

func TestEditValidates(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	it, _ := s.Add("a")
	before, _ := os.ReadFile(f.file)
	bad := []Edit{
		{Text: "a", Time: "14:00"},
		{Text: "a", Date: "11/11"},
		{Text: "a", Date: "2026-11-11", Time: "25:00"},
		{Text: "a", Repeat: &Repeat{Kind: "daily"}},
		{Text: "a", Date: "2026-11-11", Repeat: &Repeat{Kind: "weekly", Weekday: 9}},
		{Text: "  "},
	}
	for _, e := range bad {
		if _, err := s.Edit(it.ID, e); err == nil {
			t.Errorf("%+v accepted", e)
		}
	}
	after, _ := os.ReadFile(f.file)
	if string(before) != string(after) {
		t.Fatal("file changed by a rejected edit")
	}
	got, err := s.Edit(it.ID, Edit{Text: " b ", Date: "2026-11-13", Time: "14:00", Repeat: &Repeat{Kind: "weekly", Weekday: 5}})
	if err != nil {
		t.Fatal(err)
	}
	if got.Text != "b" || got.Date != "2026-11-13" || got.Time != "14:00" || got.Repeat == nil || got.Repeat.Weekday != 5 {
		t.Fatalf("got %+v", got)
	}
	if d := f.onDisk(t).Items[0]; d.Date != "2026-11-13" || d.Repeat == nil {
		t.Fatalf("on disk %+v", d)
	}
	if _, err := s.Edit("nope", Edit{Text: "x"}); err == nil {
		t.Fatal("unknown id accepted")
	}
}

func TestCheckPlainItem(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	it, _ := s.Add("a")
	if err := s.Check(it.ID); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, s).Items[0]; !got.DoneAt.Equal(f.now) {
		t.Fatalf("doneAt = %v", got.DoneAt)
	}
	if err := s.Uncheck(it.ID); err != nil {
		t.Fatal(err)
	}
	if got := mustList(t, s).Items[0]; !got.DoneAt.IsZero() {
		t.Fatalf("doneAt = %v", got.DoneAt)
	}
	s.Check(it.ID)
	f.now = f.now.AddDate(0, 0, 1)
	err := s.Uncheck(it.ID)
	if err == nil || err.Error() != "só dá para desmarcar o que foi feito hoje" {
		t.Fatalf("err = %v", err)
	}
}

func TestCheckRepeatingItemAdvances(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	it, _ := s.Add("check-in")
	s.Edit(it.ID, Edit{Text: "check-in", Date: "2026-11-13", Repeat: &Repeat{Kind: "weekly", Weekday: 5}})
	if err := s.Check(it.ID); err != nil {
		t.Fatal(err)
	}
	n := mustList(t, s)
	if n.Items[0].Date != "2026-11-20" || !n.Items[0].DoneAt.IsZero() {
		t.Fatalf("item %+v", n.Items[0])
	}
	if len(n.Done) != 1 {
		t.Fatalf("done %+v", n.Done)
	}
	d := n.Done[0]
	if d.ItemID != it.ID || d.Text != "check-in" || !d.DoneAt.Equal(f.now) || d.PrevDate != "2026-11-13" || d.ID == "" {
		t.Fatalf("done %+v", d)
	}
}

func TestCheckOverdueRepeatingItem(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	it, _ := s.Add("prs")
	s.Edit(it.ID, Edit{Text: "prs", Date: "2026-11-08", Repeat: &Repeat{Kind: "daily"}})
	s.Check(it.ID)
	n := mustList(t, s)
	if n.Items[0].Date != "2026-11-12" || len(n.Done) != 1 {
		t.Fatalf("got %+v", n)
	}
}

func TestUncheckDoneRestoresTheDate(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	it, _ := s.Add("prs")
	s.Edit(it.ID, Edit{Text: "prs", Date: "2026-11-11", Repeat: &Repeat{Kind: "daily"}})
	s.Check(it.ID)
	done := mustList(t, s).Done[0]
	if err := s.Uncheck(done.ID); err != nil {
		t.Fatal(err)
	}
	n := mustList(t, s)
	if len(n.Done) != 0 || n.Items[0].Date != "2026-11-11" {
		t.Fatalf("got %+v", n)
	}
}

func TestDeleteItemOrDone(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	a, _ := s.Add("a")
	b, _ := s.Add("b")
	s.Edit(b.ID, Edit{Text: "b", Date: "2026-11-11", Repeat: &Repeat{Kind: "daily"}})
	s.Check(b.ID)
	done := mustList(t, s).Done[0]
	if err := s.Delete(done.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	n := mustList(t, s)
	if len(n.Done) != 0 || len(n.Items) != 1 || n.Items[0].ID != b.ID {
		t.Fatalf("got %+v", n)
	}
	if err := s.Delete("nope"); err == nil {
		t.Fatal("unknown id accepted")
	}
}

func TestStoreRereadsBeforeEachChange(t *testing.T) {
	f := newFixture(t)
	s := f.open(t)
	s.Add("a")
	n := f.onDisk(t)
	n.Items = append(n.Items, Item{ID: "hand", Text: "à mão"})
	data, _ := json.Marshal(n)
	os.WriteFile(f.file, data, 0o600)
	s.Add("b")
	var texts []string
	for _, it := range f.onDisk(t).Items {
		texts = append(texts, it.Text)
	}
	if strings.Join(texts, ",") != "a,à mão,b" {
		t.Fatalf("texts = %v", texts)
	}
}

func TestStoreMovesABrokenFileAsideOnAnyRead(t *testing.T) {
	f := newFixture(t)
	os.WriteFile(f.file, []byte("{oops"), 0o600)
	s, err := Open(f.file, func() time.Time { return f.now }, func() string { f.ids++; return fmt.Sprintf("n%d", f.ids) })
	aside := f.file + ".broken-20261111-100000"
	if err == nil || !strings.Contains(err.Error(), aside) {
		t.Fatalf("err = %v", err)
	}
	if data, _ := os.ReadFile(aside); string(data) != "{oops" {
		t.Fatalf("aside = %q", data)
	}
	if n := mustList(t, s); len(n.Items) != 0 {
		t.Fatalf("got %+v", n)
	}
	if p := s.Problems(); len(p) != 0 {
		t.Fatalf("Open's own error repeated as a problem: %v", p)
	}

	f.now = f.now.Add(time.Minute)
	os.WriteFile(f.file, []byte("{again"), 0o600)
	if _, err := s.Add("x"); err != nil {
		t.Fatal(err)
	}
	second := f.file + ".broken-20261111-100100"
	if data, _ := os.ReadFile(second); string(data) != "{again" {
		t.Fatalf("second aside = %q", data)
	}
	if got := f.onDisk(t).Items; len(got) != 1 || got[0].Text != "x" {
		t.Fatalf("on disk %+v", got)
	}
	p := s.Problems()
	if len(p) != 1 || !strings.Contains(p[0], second) {
		t.Fatalf("problems = %v", p)
	}
	if p := s.Problems(); len(p) != 0 {
		t.Fatalf("problems not drained: %v", p)
	}
}

func TestStoreMovesAsideAFileWithUnknownKeys(t *testing.T) {
	f := newFixture(t)
	os.WriteFile(f.file, []byte(`{"itens":[{"id":"x","text":"hand typed"}],"done":[]}`), 0o600)
	_, err := Open(f.file, func() time.Time { return f.now }, func() string { f.ids++; return fmt.Sprintf("n%d", f.ids) })
	aside := f.file + ".broken-20261111-100000"
	if err == nil || !strings.Contains(err.Error(), aside) {
		t.Fatalf("err = %v", err)
	}
	if data, _ := os.ReadFile(aside); !strings.Contains(string(data), "hand typed") {
		t.Fatalf("aside = %q", data)
	}
}

func TestStoreGivesHandAddedNotesAnID(t *testing.T) {
	f := newFixture(t)
	os.WriteFile(f.file, []byte(`{"items":[{"text":"a"},{"text":"b"}],"done":[]}`), 0o600)
	s := f.open(t)
	first := mustList(t, s)
	if first.Items[0].ID == "" || first.Items[1].ID == "" || first.Items[0].ID == first.Items[1].ID {
		t.Fatalf("ids %+v", first.Items)
	}
	if again := mustList(t, s); again.Items[0].ID != first.Items[0].ID {
		t.Fatalf("ids changed between reads: %s → %s", first.Items[0].ID, again.Items[0].ID)
	}
	if err := s.Check(first.Items[1].ID); err != nil {
		t.Fatal(err)
	}
}
