package agent

import (
	"context"
	"strings"
	"sync"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
)

func TestRunTaggedCoTWithStreamCallbacks(t *testing.T) {
	stepOne := thinkingOpenTag + "Consider pivot choice." + thinkingCloseTag + "\n" + defaultContinueTrigger
	stepTwo := thinkingOpenTag + "Use quicksort." + thinkingCloseTag + "\nSorted via quicksort."
	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{Content: stepOne},
			{Content: stepTwo},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})
	cfg := &taijicfg.Config{Agent: taijicfg.AgentConfig{ChainOfThought: taijicfg.ChainOfThoughtPrompt}}

	var mu sync.Mutex
	var answerDeltas, thinkDeltas []string
	thoughtStarts := 0

	_, err := Run(context.Background(), RunRequest{
		Goal:      "Explain step by step how to sort an array",
		Providers: providers,
		Tools:     tools.NewRegistry(),
		Config:    cfg,
		SubRun:    SubRunConfig{SkipTurnRouter: true},
		StreamDelta: func(delta string) error {
			mu.Lock()
			answerDeltas = append(answerDeltas, delta)
			mu.Unlock()
			return nil
		},
		StreamThinkDelta: func(delta string) error {
			mu.Lock()
			thinkDeltas = append(thinkDeltas, delta)
			mu.Unlock()
			return nil
		},
		StreamThoughtStart: func(int) error {
			thoughtStarts++
			return nil
		},
		EventHandler: func(ev events.Event) {
			if ev.Type == events.TypeContinue {
				thoughtStarts++ // also exercised via events
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if seq.calls < 2 {
		t.Fatalf("calls=%d", seq.calls)
	}
	if len(answerDeltas) == 0 && len(thinkDeltas) == 0 {
		t.Fatal("expected streamed tagged CoT output")
	}
	if !strings.Contains(strings.Join(answerDeltas, ""), "quicksort") &&
		!strings.Contains(strings.Join(thinkDeltas, ""), "quicksort") {
		t.Fatalf("deltas answer=%q think=%q", strings.Join(answerDeltas, ""), strings.Join(thinkDeltas, ""))
	}
}
