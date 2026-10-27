package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestCompleteWithContinuesChainsOnTrigger(t *testing.T) {
	calls := 0
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		calls++
		if calls == 1 {
			return &provider.CompletionResponse{Content: "step one\n<|continue|>"}, nil
		}
		return &provider.CompletionResponse{Content: "final"}, nil
	}}
	cfg := continueConfig{trigger: "<|continue|>", max: 2, prompt: "continue"}
	var types []events.Type
	emit := func(ev events.Event) { types = append(types, ev.Type) }
	msgs, resp, err := completeWithContinues(
		context.Background(), p, nil, nil, RunRequest{Goal: "plan"},
		1, cfg, completeOptions{allowContinue: true}, emit, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "final" || len(msgs) < 1 {
		t.Fatalf("resp=%q msgs=%d calls=%d", resp.Content, len(msgs), calls)
	}
	foundContinue := false
	for _, ty := range types {
		if ty == events.TypeContinue {
			foundContinue = true
		}
	}
	if !foundContinue {
		t.Fatalf("events=%v", types)
	}
}
