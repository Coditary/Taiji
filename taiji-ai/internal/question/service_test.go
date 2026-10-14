package question

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubPrompter struct {
	answers []Answer
	err     error
}

func (s stubPrompter) Prompt(context.Context, Request) ([]Answer, error) {
	return s.answers, s.err
}

func TestServiceAskWithPrompter(t *testing.T) {
	svc := NewService(stubPrompter{answers: []Answer{{"A"}, {"B", "C"}}})
	answers, err := svc.Ask(context.Background(), AskInput{
		Questions: []Info{
			{Question: "Q1", Options: []Option{{Label: "A"}, {Label: "B"}}},
			{Question: "Q2", Options: []Option{{Label: "B"}, {Label: "C"}}, Multiple: true},
		},
	})
	if err != nil {
		t.Fatalf("Ask: %v", err)
	}
	if len(answers) != 2 || answers[0][0] != "A" || len(answers[1]) != 2 {
		t.Fatalf("unexpected answers: %#v", answers)
	}
}

func TestServiceReplyDeferred(t *testing.T) {
	svc := NewService(nil)
	done := make(chan []Answer, 1)
	go func() {
		answers, err := svc.Ask(context.Background(), AskInput{
			Questions: []Info{{Question: "Pick", Options: []Option{{Label: "A"}, {Label: "B"}}}},
		})
		if err != nil {
			t.Errorf("Ask: %v", err)
			done <- nil
			return
		}
		done <- answers
	}()

	pending := waitPending(t, svc, 1)
	if err := svc.Reply(ReplyInput{RequestID: pending[0].ID, Answers: []Answer{{"A"}}}); err != nil {
		t.Fatalf("Reply: %v", err)
	}
	answers := <-done
	if len(answers) != 1 || answers[0][0] != "A" {
		t.Fatalf("answers = %#v", answers)
	}
}

func TestServiceRejectDeferred(t *testing.T) {
	svc := NewService(nil)
	errCh := make(chan error, 1)
	go func() {
		_, err := svc.Ask(context.Background(), AskInput{
			Questions: []Info{{Question: "Pick", Options: []Option{{Label: "A"}, {Label: "B"}}}},
		})
		errCh <- err
	}()
	pending := waitPending(t, svc, 1)
	if err := svc.Reject(pending[0].ID); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if err := <-errCh; !errors.Is(err, ErrRejected) {
		t.Fatalf("err = %v, want ErrRejected", err)
	}
}

func waitPending(t *testing.T, svc *Service, want int) []Request {
	t.Helper()
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		if pending := svc.ListPending(); len(pending) == want {
			return pending
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("pending = %d, want %d", len(svc.ListPending()), want)
	return nil
}

func TestParseQuestions(t *testing.T) {
	qs, err := ParseQuestions(map[string]any{
		"questions": []any{
			map[string]any{
				"question": "Deploy where?",
				"options": []any{
					map[string]any{"label": "Staging"},
					map[string]any{"label": "Production"},
				},
				"multiple": true,
			},
		},
	})
	if err != nil {
		t.Fatalf("ParseQuestions: %v", err)
	}
	if len(qs) != 1 || !qs[0].Multiple || qs[0].Header == "" {
		t.Fatalf("unexpected questions: %#v", qs)
	}
}
