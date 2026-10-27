package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
)

type askUserStub struct{}

func (askUserStub) Name() string               { return "ask_user" }
func (askUserStub) Description() string        { return "ask" }
func (askUserStub) Parameters() map[string]any { return map[string]any{"type": "object"} }
func (askUserStub) Run(context.Context, map[string]any) (string, error) {
	return "question sent", nil
}

func TestRunAskUserToolEmitsPromptEvent(t *testing.T) {
	reg := tools.NewRegistry()
	reg.Register(askUserStub{})

	seq := &sequenceProvider{
		id: "seq",
		responses: []provider.CompletionResponse{
			{ToolCalls: []provider.ToolCall{{ID: "a1", Name: "ask_user", Args: `{"questions":["Ok?"]}`}}},
			{Content: "done"},
		},
	}
	providers := provider.NewStaticRegistry("seq", map[string]provider.Provider{"seq": seq})

	var sawAsk bool
	_, err := Run(context.Background(), RunRequest{
		Goal:      "ask me something",
		Providers: providers,
		Tools:     reg,
		WorkDir:   t.TempDir(),
		SubRun:    SubRunConfig{SkipTurnRouter: true},
		EventHandler: func(ev events.Event) {
			if ev.Type == events.TypeUserPrompt && ev.IsAskUser {
				sawAsk = true
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = sawAsk
	out := executeToolCalls(context.Background(), nil, reg, hooks.Env{}, []provider.ToolCall{
		{ID: "x", Name: "ask_user", Args: `{}`},
	}, nil)
	emitAskUserToolActivity(func(events.Event) {}, 1, "sess", provider.ToolCall{Name: "ask_user"}, out[0])
	if len(out) != 1 {
		t.Fatal("expected ask_user execution")
	}
}
