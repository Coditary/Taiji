package agent

import (
	"context"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
)

func testDummyRegistry(t *testing.T) *provider.Registry {
	cfg := &taijicfg.Config{
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
	return reg
}

func TestRunDummyProviderInputOutputAlpha(t *testing.T) {
	reg := testDummyRegistry(t)
	for _, tc := range []struct {
		in, want string
	}{
		{"Hallo", "Hallo"},
		{"Ping", "Pong"},
	} {
		result, err := Run(context.Background(), RunRequest{
			Goal:       tc.in,
			ProviderID: "dummy_a",
			Providers:  reg,
			Tools:      tools.NewRegistry(),
			Config:     &taijicfg.Config{},
			SubRun:     SubRunConfig{SkipTurnRouter: true},
		})
		if err != nil {
			t.Fatalf("%q: %v", tc.in, err)
		}
		if result.Answer != tc.want {
			t.Fatalf("%q: answer=%q want %q", tc.in, result.Answer, tc.want)
		}
	}
}

func TestRunDummyProviderSwitchChangesOutput(t *testing.T) {
	reg := testDummyRegistry(t)
	toolsReg := tools.NewRegistry()
	cfg := &taijicfg.Config{}

	run := func(providerID, line string) string {
		result, err := Run(context.Background(), RunRequest{
			Goal:       line,
			ProviderID: providerID,
			Providers:  reg,
			Tools:      toolsReg,
			Config:     cfg,
			SubRun:     SubRunConfig{SkipTurnRouter: true},
		})
		if err != nil {
			t.Fatalf("provider=%s line=%q: %v", providerID, line, err)
		}
		return result.Answer
	}

	if got := run("dummy_a", "Hallo"); got != "Hallo" {
		t.Fatalf("dummy_a Hallo: %q", got)
	}
	if got := run("dummy_b", "Hallo"); got != "Servus" {
		t.Fatalf("dummy_b Hallo: %q", got)
	}
	if got := run("dummy_a", "Ping"); got != "Pong" {
		t.Fatalf("dummy_a Ping: %q", got)
	}
	if got := run("dummy_b", "Ping"); got != "Bing" {
		t.Fatalf("dummy_b Ping: %q", got)
	}
}
