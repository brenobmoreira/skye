package tmuxctl

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`(\d+)\.(\d+)`)

func ParseVersion(s string) (int, int, error) {
	m := versionPattern.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, fmt.Errorf("unrecognized tmux version %q", strings.TrimSpace(s))
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	return major, minor, nil
}

func CheckVersion() error {
	out, err := exec.Command("tmux", "-V").Output()
	if err != nil {
		return fmt.Errorf("tmux não encontrado: instale com `sudo apt install tmux` (%w)", err)
	}
	major, minor, err := ParseVersion(string(out))
	if err != nil {
		return err
	}
	if major < 3 || (major == 3 && minor < 2) {
		return fmt.Errorf("skye precisa de tmux ≥ 3.2; encontrado %s", strings.TrimSpace(string(out)))
	}
	return nil
}
