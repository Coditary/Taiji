package session

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestEncodeDecodeActorChromeName(t *testing.T) {
	chrome := ActorChrome{TemplateID: "zwerg-schmied", DisplayName: "Zwergenschmied", Avatar: "⛏"}
	encoded := EncodeActorChromeName(chrome)
	if encoded == "" {
		t.Fatal("expected encoded name")
	}
	got, ok := DecodeActorChromeName(encoded)
	if !ok {
		t.Fatal("decode failed")
	}
	if got.DisplayName != chrome.DisplayName || got.Avatar != chrome.Avatar {
		t.Fatalf("round trip = %#v", got)
	}
	if got.AuthorLabel() != "⛏ Zwergenschmied" {
		t.Fatalf("label = %q", got.AuthorLabel())
	}
}

func TestStampLastAssistantActor(t *testing.T) {
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "hi"},
		{Role: provider.RoleAssistant, Content: "hello"},
	}
	st := Settings{TemplateID: "story", TemplateDisplayName: "Story", Avatar: "📖"}
	StampLastAssistantActor(msgs, st)
	chrome, ok := DecodeActorChromeName(msgs[1].Name)
	if !ok || chrome.TemplateID != "story" {
		t.Fatalf("stamped = %#v ok=%v", msgs[1].Name, ok)
	}
}
