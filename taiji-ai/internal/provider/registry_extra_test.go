package provider

import (
	"context"
	"strings"
	"testing"
)

type stubProvider struct {
	id, typ string
}

func (s stubProvider) ID() string   { return s.id }
func (s stubProvider) Type() string { return s.typ }
func (s stubProvider) Complete(context.Context, CompletionRequest) (*CompletionResponse, error) {
	return &CompletionResponse{Content: "ok"}, nil
}

func TestStaticRegistryListDefaultGet(t *testing.T) {
	reg := NewStaticRegistry("a", map[string]Provider{
		"a": stubProvider{id: "a", typ: "dummy"},
		"b": stubProvider{id: "b", typ: "openai"},
	})
	def, err := reg.Default()
	if err != nil || def.ID() != "a" {
		t.Fatalf("default: err=%v id=%q", err, def.ID())
	}
	list := reg.List()
	if len(list) != 2 {
		t.Fatalf("list=%v", list)
	}
	_, err = reg.Get("missing")
	if err == nil || !strings.Contains(err.Error(), "not found") {
		t.Fatalf("get missing: %v", err)
	}
	regEmpty := NewStaticRegistry("", map[string]Provider{})
	if _, err := regEmpty.Default(); err == nil {
		t.Fatal("expected no default error")
	}
}

func TestJoinThinkingBlocks(t *testing.T) {
	if JoinThinkingBlocks(nil) != "" {
		t.Fatal("nil blocks")
	}
	got := JoinThinkingBlocks([]ThinkingBlock{
		{Text: " first "},
		{Text: ""},
		{Text: "second"},
	})
	if got != "first\n\nsecond" {
		t.Fatalf("got %q", got)
	}
}

func TestParseToolChoiceBridge(t *testing.T) {
	if string(parseToolChoice("required")) != "required" {
		t.Fatalf("choice=%q", parseToolChoice("required"))
	}
}

func TestDummyProviderIDType(t *testing.T) {
	p := stubProvider{id: "x", typ: "dummy"}
	if p.ID() != "x" || p.Type() != "dummy" {
		t.Fatalf("id/type mismatch")
	}
}
