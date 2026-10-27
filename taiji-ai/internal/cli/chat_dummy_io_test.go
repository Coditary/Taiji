package cli

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/agent"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func newDummyChatState(t *testing.T) *chatState {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
			"dummy_b": {Type: "dummy", Model: "beta"},
		},
	}
	reg, err := provider.NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := session.New(root, "", "dummy_a", "chat")
	if err != nil {
		t.Fatal(err)
	}
	return &chatState{
		app:        &App{Config: cfg},
		providerID: "dummy_a",
		session:    sess,
		workDir:    root,
		rt: &runtime.Runtime{
			Config:    cfg,
			Providers: reg,
			Tools:     tools.NewRegistry(),
		},
	}
}

func runChatTurn(state *chatState, line string) (string, error) {
	result, err := agent.Run(context.Background(), agent.RunRequest{
		Goal:       line,
		ProviderID: state.providerID,
		Session:    state.session,
		Providers:  state.rt.Providers,
		Tools:      state.rt.Tools,
		Config:     state.app.Config,
		WorkDir:    state.workDir,
		SubRun:     agent.SubRunConfig{SkipTurnRouter: true},
	})
	if err != nil {
		return "", err
	}
	return result.Answer, nil
}

func TestChatDummyProviderInputOutput(t *testing.T) {
	state := newDummyChatState(t)
	answer, err := runChatTurn(state, "Ping")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "Pong" {
		t.Fatalf("answer=%q", answer)
	}
}

func TestChatProviderSwitchUsesDummyBeta(t *testing.T) {
	state := newDummyChatState(t)

	handled, exit, err := state.handleSlashCommand("/provider dummy_b")
	if err != nil {
		t.Fatal(err)
	}
	if !handled || exit {
		t.Fatalf("handled=%v exit=%v", handled, exit)
	}
	if state.providerID != "dummy_b" {
		t.Fatalf("providerID=%q", state.providerID)
	}

	answer, err := runChatTurn(state, "Ping")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "Bing" {
		t.Fatalf("after switch answer=%q want Bing", answer)
	}

	answer, err = runChatTurn(state, "Hallo")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "Servus" {
		t.Fatalf("Hallo on beta=%q want Servus", answer)
	}
}
