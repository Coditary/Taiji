package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestCompleteWithRetryEventuallyAfterEmptyResponses(t *testing.T) {
	attempts := 0
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		attempts++
		if attempts < 3 {
			return nil, errEmptyResponse
		}
		return &provider.CompletionResponse{Content: "recovered"}, nil
	}}
	resp, err := completeWithRetry(context.Background(), p, RunRequest{}, nil, nil, 1, false, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "recovered" || attempts != 3 {
		t.Fatalf("content=%q attempts=%d", resp.Content, attempts)
	}
}

func TestCompleteWithContinuesWithStreamDelta(t *testing.T) {
	p := &sequenceProvider{
		id:        "seq",
		responses: []provider.CompletionResponse{{Content: "streamed answer"}},
	}
	var deltas []string
	_, _, err := completeWithContinues(
		context.Background(), p,
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{
			Goal: "hi",
			StreamDelta: func(delta string) error {
				deltas = append(deltas, delta)
				return nil
			},
		},
		1,
		continueConfig{},
		completeOptions{},
		func(events.Event) {},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(deltas) == 0 {
		t.Fatal("expected stream deltas")
	}
}
