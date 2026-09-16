package session

import (
	"testing"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func TestResolveContextWindowFromProviderConfig(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"local": {ContextWindow: 32_768, Model: "llama3"},
		},
	}
	if got := ResolveContextWindow(cfg, "local"); got != 32_768 {
		t.Fatalf("ResolveContextWindow = %d, want 32768", got)
	}
}

func TestResolveContextWindowFromModelName(t *testing.T) {
	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"cloud": {Model: "claude-sonnet-4"},
		},
	}
	if got := ResolveContextWindow(cfg, "cloud"); got != 200_000 {
		t.Fatalf("ResolveContextWindow = %d, want 200000", got)
	}
}

func TestContextSnapshotUsesLastInputTokens(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "ctx1", "cloud", "chat")
	if err != nil {
		t.Fatal(err)
	}
	sess.State.Messages = []provider.Message{
		{Role: provider.RoleUser, Content: "hello"},
	}
	sess.State.Stats.LastInputTokens = 12_345
	sess.State.Stats.EstimatedCostUSD = 0.0123

	cfg := &config.Config{
		Providers: map[string]config.ProviderConfig{
			"cloud": {ContextWindow: 100_000},
		},
	}
	snap := sess.ContextSnapshot(cfg, "cloud")
	if snap.ContextTokens != 12_345 {
		t.Fatalf("ContextTokens = %d", snap.ContextTokens)
	}
	if snap.ContextWindow != 100_000 {
		t.Fatalf("ContextWindow = %d", snap.ContextWindow)
	}
	if snap.ContextUsedPct < 12.3 || snap.ContextUsedPct > 12.4 {
		t.Fatalf("ContextUsedPct = %f", snap.ContextUsedPct)
	}
	if snap.EstimatedCostUSD != 0.0123 {
		t.Fatalf("EstimatedCostUSD = %f", snap.EstimatedCostUSD)
	}
}
