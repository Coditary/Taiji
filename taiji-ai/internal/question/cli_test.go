package question

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCLIPrompterSingleChoice(t *testing.T) {
	in := strings.NewReader("1\n")
	out := &bytes.Buffer{}
	p := CLIPrompter{In: in, Out: out}
	answers, err := p.Prompt(context.Background(), Request{
		Questions: []Info{{
			Header:   "Pick",
			Question: "One?",
			Options:  []Option{{Label: "Alpha"}, {Label: "Beta"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 1 || answers[0][0] != "Alpha" {
		t.Fatalf("answers=%v", answers)
	}
}

func TestCLIPrompterRejectCancel(t *testing.T) {
	in := strings.NewReader("cancel\n")
	p := CLIPrompter{In: in, Out: ioDiscard()}
	_, err := p.Prompt(context.Background(), Request{
		Questions: []Info{{
			Header:   "X",
			Question: "Y?",
			Options:  []Option{{Label: "A"}},
		}},
	})
	if err != ErrRejected {
		t.Fatalf("err=%v", err)
	}
}

func TestCLIPrompterInvalidQuestions(t *testing.T) {
	p := CLIPrompter{In: strings.NewReader(""), Out: ioDiscard()}
	_, err := p.Prompt(context.Background(), Request{})
	if err != ErrInvalidQuestions {
		t.Fatalf("err=%v", err)
	}
}

func ioDiscard() *bytes.Buffer { return &bytes.Buffer{} }
