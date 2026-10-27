package question

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestCLIPrompterMultiSelectAndSummary(t *testing.T) {
	in := strings.NewReader("1, 2\n1\n\n")
	out := &bytes.Buffer{}
	p := CLIPrompter{In: in, Out: out}
	answers, err := p.Prompt(context.Background(), Request{
		Questions: []Info{
			{
				Header:   "Pick many",
				Question: "Colors?",
				Multiple: true,
				Options:  []Option{{Label: "Red"}, {Label: "Blue"}},
			},
			{
				Header:   "Second",
				Question: "Size?",
				Options:  []Option{{Label: "S"}, {Label: "L"}},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(answers) != 2 || len(answers[0]) != 2 {
		t.Fatalf("answers=%v", answers)
	}
	if !strings.Contains(out.String(), "Summary:") {
		t.Fatal("expected summary block")
	}
}

func TestCLIPrompterCustomSingle(t *testing.T) {
	in := strings.NewReader("2\nMy custom\n")
	out := &bytes.Buffer{}
	p := CLIPrompter{In: in, Out: out}
	answers, err := p.Prompt(context.Background(), Request{
		Questions: []Info{{
			Header:   "Custom",
			Question: "Pick?",
			Options:  []Option{{Label: "Preset"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if answers[0][0] != "My custom" {
		t.Fatalf("answers=%v", answers)
	}
}

func TestCLIPrompterEmptyAnswerRejected(t *testing.T) {
	p := CLIPrompter{In: strings.NewReader("\n"), Out: ioDiscard()}
	_, err := p.Prompt(context.Background(), Request{
		Questions: []Info{{Header: "X", Question: "Y?", Options: []Option{{Label: "A"}}}},
	})
	if err == nil {
		t.Fatal("expected error for empty answer")
	}
}

func TestRejectPrompter(t *testing.T) {
	var rp RejectPrompter
	_, err := rp.Prompt(context.Background(), Request{
		Questions: []Info{{Header: "X", Question: "Y?", Options: []Option{{Label: "A"}}}},
	})
	if err != ErrRejected {
		t.Fatalf("err=%v", err)
	}
}

func TestInfoAllowsCustom(t *testing.T) {
	falseVal := false
	if (Info{Custom: &falseVal}).AllowsCustom() {
		t.Fatal("custom disabled")
	}
	if !(Info{}).AllowsCustom() {
		t.Fatal("default allows custom")
	}
}
