package session

import (
	"context"
	"testing"
)

func TestActiveSessionContext(t *testing.T) {
	if _, ok := ActiveSessionFrom(nil); ok {
		t.Fatal("nil ctx")
	}
	s := &Session{State: State{ID: "x"}}
	ctx := WithActiveSession(context.Background(), s)
	got, ok := ActiveSessionFrom(ctx)
	if !ok || got.State.ID != "x" {
		t.Fatalf("got=%v ok=%v", got, ok)
	}
	if WithActiveSession(nil, s) != nil {
		t.Fatal("nil ctx passthrough")
	}
}
