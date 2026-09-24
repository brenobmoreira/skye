// Package opener opens web links in the user's browser. Inside WSL that is the Windows browser,
// which xdg-open does not know about.
package opener

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"strings"
)

// Command returns the program and arguments that open rawURL, refusing anything that is not a
// plain http or https link.
func Command(rawURL string, wsl bool) (string, []string, error) {
	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || strings.ContainsAny(rawURL, "\x00\r\n\t ") {
		return "", nil, errors.New("só abro links http ou https")
	}
	if wsl {
		return "explorer.exe", []string{rawURL}, nil
	}
	return "xdg-open", []string{rawURL}, nil
}

func InWSL(getenv func(string) string, procVersion func() string) bool {
	return getenv("WSL_DISTRO_NAME") != "" || strings.Contains(strings.ToLower(procVersion()), "microsoft")
}

func readProcVersion() string {
	b, _ := os.ReadFile("/proc/version")
	return string(b)
}

// Open starts the browser without waiting; explorer.exe exits with 1 even when it worked.
func Open(rawURL string) error {
	name, args, err := Command(rawURL, InWSL(os.Getenv, readProcVersion))
	if err != nil {
		return err
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
