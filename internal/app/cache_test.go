package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/brenobmoreira/skye/internal/procs"
)

// cacheProcs answers Machine with each reading in turn.
type cacheProcs struct{ readings []procs.Machine }

func (c *cacheProcs) Machine() (procs.Machine, error) {
	m := c.readings[0]
	if len(c.readings) > 1 {
		c.readings = c.readings[1:]
	}
	return m, nil
}
func (c *cacheProcs) List() ([]procs.Proc, error) { return nil, nil }

func TestFreeCacheDropsAndCompactsThenReports(t *testing.T) {
	var ran []string
	a := New(Options{
		Procs: &cacheProcs{readings: []procs.Machine{{CacheMB: 4600, LargeFree: 0}, {CacheMB: 1200, LargeFree: 812}}},
		RunRoot: func(name string, args ...string) ([]byte, error) {
			ran = append(ran, strings.Join(append([]string{name}, args...), " "))
			return nil, nil
		},
		Emit: func(string, any) {}, Logf: t.Logf,
	})
	got, err := a.FreeCache()
	if err != nil {
		t.Fatal(err)
	}
	if got != (FreedCache{FreedMB: 3400, CacheMB: 1200, LargeFree: 812}) {
		t.Fatalf("got %+v", got)
	}
	want := []string{"sync", "sudo -n /usr/sbin/sysctl -q -w vm.drop_caches=3 vm.compact_memory=1"}
	if strings.Join(ran, "|") != strings.Join(want, "|") {
		t.Fatalf("ran %q", ran)
	}
}

func TestFreeCacheExplainsASudoThatAsksForAPassword(t *testing.T) {
	a := New(Options{
		Procs: &cacheProcs{readings: []procs.Machine{{CacheMB: 4600}}},
		RunRoot: func(name string, args ...string) ([]byte, error) {
			if name == "sudo" {
				return []byte("sudo: a password is required\n"), errors.New("exit status 1")
			}
			return nil, nil
		},
		User: "breno", Emit: func(string, any) {}, Logf: t.Logf,
	})
	_, err := a.FreeCache()
	if err == nil || !strings.Contains(err.Error(), "breno ALL=(root) NOPASSWD: /usr/sbin/sysctl -q -w vm.drop_caches=3 vm.compact_memory=1") {
		t.Fatalf("err = %v", err)
	}
}
