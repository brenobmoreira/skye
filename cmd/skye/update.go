package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/brenobmoreira/skye/internal/update"
)

// repoDir is the checkout this binary was built from, set by make build (-X main.repoDir).
var repoDir string

// executable is the installed binary, with symlinks resolved, so an update replaces the file
// itself and not a link to it.
func executable() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(self)
}

func runUpdate(args []string) int {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "uso: skye update")
		return 2
	}
	self, err := executable()
	if err != nil {
		fmt.Fprintln(os.Stderr, "skye update:", err)
		return 1
	}
	fmt.Println("compilando em", repoDir, "…")
	if err := update.Run(update.Options{Repo: repoDir, Target: self}); err != nil {
		fmt.Fprintln(os.Stderr, "skye update:", err)
		return 1
	}
	pids := webServers("/proc", os.Getpid())
	for _, pid := range pids {
		_ = syscall.Kill(pid, syscall.SIGHUP)
	}
	if len(pids) == 0 {
		fmt.Println("instalada em", self, "; nenhum skye web rodando")
	} else {
		fmt.Println("instalada em", self, "; o skye web reinicia sozinho e as janelas recarregam")
	}
	return 0
}

// webServers finds the running "skye web" processes, other than self.
func webServers(proc string, self int) []int {
	entries, err := os.ReadDir(proc)
	if err != nil {
		return nil
	}
	var out []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil || pid == self {
			continue
		}
		cmdline, err := os.ReadFile(filepath.Join(proc, e.Name(), "cmdline"))
		if err != nil {
			continue
		}
		args := strings.Split(strings.TrimRight(string(cmdline), "\x00"), "\x00")
		if len(args) >= 2 && filepath.Base(args[0]) == "skye" && args[1] == "web" {
			out = append(out, pid)
		}
	}
	return out
}
