package hooks

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseReply(t *testing.T) {
	long := strings.Repeat("a", 200)
	cases := []struct {
		name     string
		in       string
		line     string
		question bool
		options  []Option
	}{
		{name: "question", in: "Rodei os testes.\n\nFaço o push?", line: "Faço o push?", question: true},
		{name: "bold question with trailing blank lines", in: "Feito.\n\n**Posso começar?**\n\n", line: "Posso começar?", question: true},
		{name: "plain question", in: "Quer que eu levante a latência antes de fechar o valor?", line: "Quer que eu levante a latência antes de fechar o valor?", question: true},
		{
			name: "numbered choices after a question",
			in:   "Publico essa review no Gitea como comentário no PR?\n1. Publicar tudo (sai como **COMMENT**, porque não há nenhum finding crítico)\n2. Só críticos\n3. Cancelar (não publica)",
			line: "Publico essa review no Gitea como comentário no PR?",
			options: []Option{
				{N: 1, Label: "Publicar tudo"},
				{N: 2, Label: "Só críticos"},
				{N: 3, Label: "Cancelar"},
			},
		},
		{
			name:    "question in its own paragraph above the choices",
			in:      "Como prefere seguir?\n\n1. Publicar tudo\n2. Só críticos\n3. Cancelar",
			line:    "Como prefere seguir?",
			options: []Option{{N: 1, Label: "Publicar tudo"}, {N: 2, Label: "Só críticos"}, {N: 3, Label: "Cancelar"}},
		},
		{
			name:    "the question closest to the choices names them",
			in:      "Rodei a review.\nComo prefere seguir?\n\n1. Publicar\n2. Cancelar",
			line:    "Como prefere seguir?",
			options: []Option{{N: 1, Label: "Publicar"}, {N: 2, Label: "Cancelar"}},
		},
		{name: "five items are findings, not a choice", in: "Achados:\n1. a\n2. b\n3. c\n4. d\n5. e", line: "Achados:"},
		{name: "list then question is a question", in: "Plano:\n1. A\n2. B\n\nAlgum ajuste antes de eu seguir?", line: "Algum ajuste antes de eu seguir?", question: true},
		{name: "statement", in: "Instalei e conferi o binário.", line: "Instalei e conferi o binário."},
		{name: "ends in a code block", in: "Rode:\n\n```bash\nmake build\n```", line: "make build"},
		{name: "long line is cut", in: long, line: strings.Repeat("a", 119) + "…"},
		{name: "empty", in: ""},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ParseReply(c.in)
			if got.Line != c.line || got.Question != c.question || !reflect.DeepEqual(got.Options, c.options) {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

func TestParseReplyKeepsTheWholeClosingBlockForTheTooltip(t *testing.T) {
	got := ParseReply("Intro.\n\nPrimeira linha.\nSegunda linha?")
	if got.Full != "Primeira linha.\nSegunda linha?" || !got.Question {
		t.Fatalf("got %+v", got)
	}
	if long := ParseReply(strings.Repeat("b", 900)); len([]rune(long.Full)) != 600 {
		t.Fatalf("full has %d runes", len([]rune(long.Full)))
	}
}

func TestParseReplyTooltipKeepsTheQuestionAboveTheChoices(t *testing.T) {
	got := ParseReply("Feito.\n\nComo prefere seguir?\n\n1. Publicar tudo\n2. Cancelar")
	if got.Full != "Como prefere seguir?\n1. Publicar tudo\n2. Cancelar" {
		t.Fatalf("full = %q", got.Full)
	}
}
