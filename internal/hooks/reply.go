package hooks

import (
	"regexp"
	"strconv"
	"strings"
)

// How much of the closing block the list shows: the line under the item, the tooltip, and the
// label of each numbered choice.
const (
	ReplyLineLimit   = 120
	ReplyFullLimit   = 600
	OptionLabelLimit = 40
)

// Option is one numbered choice the agent closed its turn with.
type Option struct {
	N     int    `json:"n"`
	Label string `json:"label"`
}

// Reply is what the list needs from the agent's last reply: its closing block, and whether that
// block asks a yes-or-no question or offers numbered choices.
type Reply struct {
	Line     string
	Full     string
	Question bool
	Options  []Option
}

var item = regexp.MustCompile(`^(\d)[.)]\s+(.+)$`)

// ParseReply reads the closing block of a reply: the last run of non-empty lines. A closing list
// of 2 to 4 numbered items is a choice; a closing line ending in "?" is a question.
func ParseReply(text string) Reply {
	lines := closingBlock(text)
	if len(lines) == 0 {
		return Reply{}
	}
	r := Reply{Full: cut(strings.Join(lines, "\n"), ReplyFullLimit)}
	if opts, first := choices(lines); opts != nil {
		r.Options = opts
		lines = lines[:first]
		if len(lines) == 0 {
			lines = []string{opts[0].Label}
		}
	} else if last := lines[len(lines)-1]; strings.HasSuffix(last, "?") {
		r.Question = true
		r.Line = cut(last, ReplyLineLimit)
		return r
	}
	r.Line = cut(lines[0], ReplyLineLimit)
	return r
}

// closingBlock returns the cleaned lines of the last block, blocks being split by blank lines.
// Code fences are dropped so a reply ending in a code block shows its code, not the fence.
func closingBlock(text string) []string {
	var block, current []string
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			continue
		}
		if line == "" {
			if len(current) > 0 {
				block, current = current, nil
			}
			continue
		}
		current = append(current, clean(line))
	}
	if len(current) > 0 {
		block = current
	}
	return block
}

// choices finds the numbered list that ends the block, counted from 1; it returns the options
// and the index of the first item, or nil when the block does not end in 2 to 4 items.
func choices(lines []string) ([]Option, int) {
	first := len(lines)
	for first > 0 && item.MatchString(lines[first-1]) {
		first--
	}
	n := len(lines) - first
	if n < 2 || n > 4 {
		return nil, 0
	}
	opts := make([]Option, 0, n)
	for i, line := range lines[first:] {
		m := item.FindStringSubmatch(line)
		if num, _ := strconv.Atoi(m[1]); num != i+1 {
			return nil, 0
		}
		opts = append(opts, Option{N: i + 1, Label: cut(label(m[2]), OptionLabelLimit)})
	}
	return opts, first
}

func label(text string) string {
	for _, sep := range []string{" (", " — "} {
		if i := strings.Index(text, sep); i > 0 {
			text = text[:i]
		}
	}
	return strings.TrimSpace(text)
}

func clean(line string) string {
	return strings.TrimSpace(strings.NewReplacer("**", "", "`", "").Replace(line))
}

func cut(text string, limit int) string {
	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}
	return string(runes[:limit-1]) + "…"
}
