package tmuxctl

import "strings"

type Kind int

const (
	KindOther Kind = iota
	KindOutput
	KindBegin
	KindEnd
	KindError
	KindWindowClose
	KindExit
)

type Line struct {
	Kind   Kind
	Pane   string
	Data   []byte
	Number string
	Flags  string
	Window string
}

func ParseLine(s string) Line {
	switch {
	case strings.HasPrefix(s, "%output "):
		pane, data, _ := strings.Cut(strings.TrimPrefix(s, "%output "), " ")
		return Line{Kind: KindOutput, Pane: pane, Data: DecodeOctal(data)}
	case strings.HasPrefix(s, "%begin "):
		return block(KindBegin, s)
	case strings.HasPrefix(s, "%end "):
		return block(KindEnd, s)
	case strings.HasPrefix(s, "%error "):
		return block(KindError, s)
	case strings.HasPrefix(s, "%window-close "), strings.HasPrefix(s, "%unlinked-window-close "):
		fields := strings.Fields(s)
		l := Line{Kind: KindWindowClose}
		if len(fields) >= 2 {
			l.Window = fields[1]
		}
		return l
	case s == "%exit" || strings.HasPrefix(s, "%exit "):
		return Line{Kind: KindExit}
	}
	return Line{Kind: KindOther}
}

func block(kind Kind, s string) Line {
	fields := strings.Fields(s)
	l := Line{Kind: kind}
	if len(fields) >= 3 {
		l.Number = fields[2]
	}
	if len(fields) >= 4 {
		l.Flags = fields[3]
	}
	return l
}

func DecodeOctal(s string) []byte {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\\' && isOctal(s, i+1) {
			out = append(out, (s[i+1]-'0')<<6|(s[i+2]-'0')<<3|(s[i+3]-'0'))
			i += 3
			continue
		}
		out = append(out, s[i])
	}
	return out
}

func isOctal(s string, from int) bool {
	if from+3 > len(s) {
		return false
	}
	for _, c := range []byte(s[from : from+3]) {
		if c < '0' || c > '7' {
			return false
		}
	}
	return true
}
