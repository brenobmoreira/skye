package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/brenobmoreira/skye/internal/winshell"
)

const (
	createNoWindow  = 0x08000000
	detachedProcess = 0x00000008
	runTimeout      = 60 * time.Second
	logTailBytes    = 4096
)

type Shell struct{}

func (s *Shell) Connect() (winshell.Endpoint, error) {
	started := false
	start := func(name string, args ...string) error {
		started = true
		return startDetached(name, args...)
	}
	ep, err := winshell.Ensure(winshell.Port(os.Getenv), run, start, winshell.Probe, gaveUp, time.Sleep)
	if err != nil && started {
		if tail := logTail(); tail != "" {
			err = fmt.Errorf("%w\n\nsaída do skye web:\n%s", err, tail)
		}
	}
	return ep, err
}

func wslEnv() []string {
	return append(os.Environ(), "WSL_UTF8=1")
}

func run(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = wslEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	out, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		return out, fmt.Errorf("sem resposta em %s", runTimeout)
	}
	return out, err
}

func startDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Env = wslEnv()
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: createNoWindow | detachedProcess | syscall.CREATE_NEW_PROCESS_GROUP,
	}
	if log, err := createLog(); err == nil {
		defer log.Close()
		cmd.Stdout = log
		cmd.Stderr = log
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

func logPath() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "skye", "web.log")
}

func createLog() (*os.File, error) {
	path := logPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	return os.Create(path)
}

func gaveUp() (string, bool) {
	return winshell.GaveUp(logTail())
}

func logTail() string {
	f, err := os.Open(logPath())
	if err != nil {
		return ""
	}
	defer f.Close()
	if info, err := f.Stat(); err == nil && info.Size() > logTailBytes {
		_, _ = f.Seek(-logTailBytes, io.SeekEnd)
	}
	data, _ := io.ReadAll(f)
	return winshell.Redact(strings.TrimSpace(string(data)))
}
