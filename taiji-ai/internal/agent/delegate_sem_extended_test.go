package agent

import (
	"context"
	"testing"

	"github.com/coditary/taiji/internal/tools/builtin"
)

func TestSetDelegateMaxConcurrentAndSubSessionContext(t *testing.T) {
	SetDelegateMaxConcurrent(2)
	ctx := builtin.WithToolSessionID(context.Background(), "parent-sess")
	newCtx, subID := subSessionContext(ctx, "parent-sess")
	if subID == "" || subID == "parent-sess" {
		t.Fatalf("subID=%q", subID)
	}
	if got := builtin.ToolSessionIDFrom(newCtx); got != subID {
		t.Fatalf("ctx session=%q sub=%q", got, subID)
	}
	SetDelegateMaxConcurrent(defaultDelegateMaxConcurrent)
}

func TestNormalizeDelegateSessionEmpty(t *testing.T) {
	if normalizeDelegateSession("  ") != "default" {
		t.Fatal("expected default session key")
	}
}
