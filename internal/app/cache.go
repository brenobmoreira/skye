package app

import (
	"errors"
	"fmt"
	"strings"
)

// The cleanup is fixed: write what is pending to disk, then have the kernel drop its disk cache
// and compact memory. Nothing is closed; it only gives the large free blocks back.
var dropCache = []string{"/usr/sbin/sysctl", "-q", "-w", "vm.drop_caches=3", "vm.compact_memory=1"}

type FreedCache struct {
	FreedMB   int `json:"freedMb"`
	CacheMB   int `json:"cacheMb"`
	LargeFree int `json:"largeFree"`
}

// FreeCache drops the WSL disk cache and compacts memory, so Windows can open programs in WSL
// again when fragmentation left no large free block. It needs sudo without a password for that
// one command.
func (a *App) FreeCache() (FreedCache, error) {
	if a.o.Procs == nil {
		return FreedCache{}, errors.New("monitor indisponível")
	}
	before, err := a.o.Procs.Machine()
	if err != nil {
		return FreedCache{}, err
	}
	if out, err := a.o.RunRoot("sync"); err != nil {
		return FreedCache{}, fmt.Errorf("sync: %v %s", err, strings.TrimSpace(string(out)))
	}
	out, err := a.o.RunRoot("sudo", append([]string{"-n"}, dropCache...)...)
	if err != nil {
		text := strings.TrimSpace(string(out))
		if strings.Contains(text, "password") || strings.Contains(text, "senha") {
			user := a.o.User
			if user == "" {
				user = "<seu usuário>"
			}
			return FreedCache{}, fmt.Errorf("o sudo pediu senha. Para liberar só este comando, rode sudo visudo -f /etc/sudoers.d/skye e adicione a linha: %s ALL=(root) NOPASSWD: %s", user, strings.Join(dropCache, " "))
		}
		return FreedCache{}, fmt.Errorf("não consegui liberar o cache: %v %s", err, text)
	}
	after, err := a.o.Procs.Machine()
	if err != nil {
		return FreedCache{}, err
	}
	return FreedCache{FreedMB: max(before.CacheMB-after.CacheMB, 0), CacheMB: after.CacheMB, LargeFree: after.LargeFree}, nil
}
