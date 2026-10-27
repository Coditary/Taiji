package agent

import (
	"context"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestResolveContinueConfigOff(t *testing.T) {
	cfg := &taijicfg.Config{Agent: taijicfg.AgentConfig{ContinueTrigger: "off"}}
	got := resolveContinueConfig(cfg)
	if got.trigger != "" {
		t.Fatalf("trigger=%q", got.trigger)
	}
}

func TestContainsContinueTriggerVariants(t *testing.T) {
	trigger := "<|continue|>"
	if !containsContinueTrigger("step\n<|continue|>", trigger) {
		t.Fatal("line trigger")
	}
	if !containsContinueTrigger("partial <|continue", trigger) {
		t.Fatal("partial token")
	}
}

func TestResponseRequestsContinueFromThinking(t *testing.T) {
	resp := &provider.CompletionResponse{
		ThinkingBlocks: []provider.ThinkingBlock{{Text: "plan\n<|continue|>"}},
	}
	if !responseRequestsContinue(resp, "<|continue|>") {
		t.Fatal("expected continue from thinking block")
	}
}

func TestCompleteWithRetryEmptyResponseThenSuccess(t *testing.T) {
	attempts := 0
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		attempts++
		if attempts < 2 {
			return nil, errEmptyResponse
		}
		return &provider.CompletionResponse{Content: "ok"}, nil
	}}
	resp, err := completeWithRetry(context.Background(), p, RunRequest{}, nil, nil, 1, false, nil, nil, nil)
	if err != nil || resp.Content != "ok" {
		t.Fatalf("resp=%+v err=%v attempts=%d", resp, err, attempts)
	}
}

func TestCompleteWithContinuesSingleAnswer(t *testing.T) {
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		return &provider.CompletionResponse{Content: "final answer"}, nil
	}}
	var emitted []events.Type
	emit := func(ev events.Event) { emitted = append(emitted, ev.Type) }
	msgs, resp, err := completeWithContinues(
		context.Background(), p, nil, nil, RunRequest{Goal: "hi"},
		1, continueConfig{trigger: "", max: 2}, completeOptions{}, emit, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(msgs) != 1 || resp.Content != "final answer" {
		t.Fatalf("msgs=%d content=%q", len(msgs), resp.Content)
	}
}

type retryStubProvider struct {
	fn func() (*provider.CompletionResponse, error)
}

func (r *retryStubProvider) ID() string   { return "stub" }
func (r *retryStubProvider) Type() string { return "stub" }
func (r *retryStubProvider) Complete(ctx context.Context, req provider.CompletionRequest) (*provider.CompletionResponse, error) {
	return r.fn()
}

var errEmptyResponse = providerCompletionEmpty("empty response from model")

type providerCompletionEmpty string

func (e providerCompletionEmpty) Error() string { return string(e) }
