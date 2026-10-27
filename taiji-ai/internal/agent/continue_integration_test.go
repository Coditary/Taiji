package agent

import (
	"context"
	"errors"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestCompleteWithRetryNonEmptyErrorStopsImmediately(t *testing.T) {
	calls := 0
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		calls++
		return nil, errors.New("network failure")
	}}
	_, err := completeWithRetry(context.Background(), p, RunRequest{}, nil, nil, 1, false, nil, nil, nil)
	if err == nil || !strings.Contains(err.Error(), "network") {
		t.Fatalf("err=%v", err)
	}
	if calls != 1 {
		t.Fatalf("calls=%d want 1", calls)
	}
}

func TestCompleteWithContinuesEmitsThoughtFromNativeBlocks(t *testing.T) {
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		return &provider.CompletionResponse{
			Content: "answer bit",
			ThinkingBlocks: []provider.ThinkingBlock{{
				Text:   "internal reasoning",
				Source: "model",
			}},
		}, nil
	}}
	var types []events.Type
	emit := func(ev events.Event) { types = append(types, ev.Type) }
	_, resp, err := completeWithContinues(
		context.Background(), p, nil, nil, RunRequest{Goal: "q"},
		1, continueConfig{}, completeOptions{}, emit, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "answer bit" {
		t.Fatalf("content=%q", resp.Content)
	}
	foundThought := false
	for _, ty := range types {
		if ty == events.TypeThought {
			foundThought = true
		}
	}
	if !foundThought {
		t.Fatalf("events=%v", types)
	}
}

func TestCompleteWithContinuesRepairsUnclosedThinking(t *testing.T) {
	calls := 0
	partial := thinkingOpenTag + "still planning"
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		calls++
		if calls == 1 {
			return &provider.CompletionResponse{Content: partial}, nil
		}
		return &provider.CompletionResponse{Content: thinkingCloseTag + " fixed answer"}, nil
	}}
	cfg := &taijicfg.Config{Agent: taijicfg.AgentConfig{ChainOfThought: taijicfg.ChainOfThoughtPrompt}}
	var deltas []string
	emit := func(events.Event) {}
	_, resp, err := completeWithContinues(
		context.Background(), p, []provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{Goal: "plan task", Config: cfg},
		1,
		continueConfig{trigger: ""},
		completeOptions{useTaggedCoT: true},
		emit,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if calls < 2 {
		t.Fatalf("expected repair completion, calls=%d", calls)
	}
	if !strings.Contains(resp.Content, "fixed answer") {
		t.Fatalf("content=%q deltas=%v", resp.Content, deltas)
	}
}

func TestCompleteWithContinuesStreamEmbeddedToolEmitsReset(t *testing.T) {
	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: "preface\n[tool call] list_dir {\"path\": \".\"}"},
		},
	}
	var types []events.Type
	emit := func(ev events.Event) { types = append(types, ev.Type) }
	var streamed []string
	_, _, err := completeWithContinues(
		context.Background(), seq, []provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{
			Goal: "list",
			StreamDelta: func(delta string) error {
				streamed = append(streamed, delta)
				return nil
			},
		},
		1,
		continueConfig{},
		completeOptions{},
		emit,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	hasReset := false
	for _, ty := range types {
		if ty == events.TypeStreamReset {
			hasReset = true
		}
	}
	if !hasReset {
		t.Fatalf("events=%v streamed=%q", types, strings.Join(streamed, ""))
	}
}

func TestEmitCompletedThinkingTaggedBlocks(t *testing.T) {
	content := thinkingOpenTag + "step one" + thinkingCloseTag + " visible"
	var texts []string
	emit := func(ev events.Event) {
		if ev.Type == events.TypeThought {
			texts = append(texts, ev.Text)
		}
	}
	emitCompletedThinking(emit, 1, &provider.CompletionResponse{Content: content}, content, "", true, RunRequest{})
	if len(texts) != 1 || !strings.Contains(texts[0], "step one") {
		t.Fatalf("thoughts=%v", texts)
	}
}

func TestRepairUnclosedThinkingDirect(t *testing.T) {
	partial := thinkingOpenTag + "open thought"
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		return &provider.CompletionResponse{Content: thinkingCloseTag + " repaired"}, nil
	}}
	streamed, resp, err := repairUnclosedThinking(
		context.Background(), p,
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{},
		partial,
		func(string) error { return nil },
	)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil || !strings.Contains(resp.Content, "repaired") {
		t.Fatalf("resp=%+v streamed=%q", resp, streamed)
	}
}
