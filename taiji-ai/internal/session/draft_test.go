package session

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestIsDraftWithSession(t *testing.T) {
	if IsDraft(nil) {
		t.Fatal("nil session is not draft")
	}
	s := &Session{State: State{Messages: []provider.Message{
		{Role: provider.RoleUser, Content: "hi"},
	}}}
	if IsDraft(s) {
		t.Fatal("has visible turn")
	}
	empty := &Session{}
	if !IsDraft(empty) {
		t.Fatal("no messages should be draft")
	}
}
