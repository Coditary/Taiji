package question

import (
	"context"
	"testing"
)

func TestAttachRunAndToolCallContext(t *testing.T) {
	ctx := AttachRun(context.Background(), RunConfig{SessionID: "sess-1"})
	if SessionIDFrom(ctx) != "sess-1" {
		t.Fatalf("session=%q", SessionIDFrom(ctx))
	}
	ctx = WithToolCall(ctx, "call-9")
	if ToolCallIDFrom(ctx) != "call-9" {
		t.Fatalf("call=%q", ToolCallIDFrom(ctx))
	}
	if SessionIDFrom(context.Background()) != "" {
		t.Fatal("empty ctx")
	}
}

func TestOnAskFromContext(t *testing.T) {
	called := false
	ctx := AttachRun(context.Background(), RunConfig{
		SessionID: "s",
		OnAsk: func(Request) {
			called = true
		},
	})
	if fn := onAskFrom(ctx); fn == nil {
		t.Fatal("expected onAsk")
	} else {
		fn(Request{})
	}
	if !called {
		t.Fatal("onAsk not invoked")
	}
}
