package agent

import (
	"context"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

func TestCompleteWithContinuesMultiThoughtChain(t *testing.T) {
	calls := 0
	stepOne := thinkingOpenTag + "first step" + thinkingCloseTag + "\n" + defaultContinueTrigger
	stepTwo := thinkingOpenTag + "second step" + thinkingCloseTag + "\nFinal conclusion."
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		calls++
		if calls == 1 {
			return &provider.CompletionResponse{Content: stepOne}, nil
		}
		return &provider.CompletionResponse{Content: stepTwo}, nil
	}}
	cfg := &taijicfg.Config{Agent: taijicfg.AgentConfig{ChainOfThought: taijicfg.ChainOfThoughtFull}}
	var continues int
	emit := func(ev events.Event) {
		if ev.Type == events.TypeContinue {
			continues++
		}
	}
	_, resp, err := completeWithContinues(
		context.Background(), p,
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{Goal: "Explain step by step how sorting works", Config: cfg},
		1,
		continueConfig{trigger: defaultContinueTrigger, max: 4, prompt: chainContinuePrompt()},
		completeOptions{allowContinue: true, useTaggedCoT: true},
		emit,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if continues < 1 {
		t.Fatalf("continues=%d calls=%d", continues, calls)
	}
	if !strings.Contains(resp.Content, "Final conclusion") {
		t.Fatalf("content=%q", resp.Content)
	}
}

func TestCompleteWithContinuesEmitsMessageWithoutStream(t *testing.T) {
	p := &retryStubProvider{fn: func() (*provider.CompletionResponse, error) {
		return &provider.CompletionResponse{Content: "plain reply"}, nil
	}}
	var types []events.Type
	emit := func(ev events.Event) { types = append(types, ev.Type) }
	_, _, err := completeWithContinues(
		context.Background(), p,
		[]provider.Message{{Role: provider.RoleSystem, Content: "sys"}},
		nil,
		RunRequest{Goal: "hi"},
		1,
		continueConfig{},
		completeOptions{},
		emit,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, ty := range types {
		if ty == events.TypeMessage {
			found = true
		}
	}
	if !found {
		t.Fatalf("events=%v", types)
	}
}
