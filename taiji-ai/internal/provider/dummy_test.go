package provider

import (
	"context"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestDummyReplyAlpha(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hallo", "Hallo"},
		{"Ping", "Pong"},
		{"foo", "foo"},
	}
	for _, tc := range tests {
		if got := dummyReply("alpha", tc.in); got != tc.want {
			t.Fatalf("alpha %q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDummyReplyBeta(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Hallo", "Servus"},
		{"Ping", "Bing"},
		{"foo", "beta:foo"},
	}
	for _, tc := range tests {
		if got := dummyReply("beta", tc.in); got != tc.want {
			t.Fatalf("beta %q: got %q want %q", tc.in, got, tc.want)
		}
	}
}

func TestDummyProviderCompleteUsesLastUserMessage(t *testing.T) {
	p, err := newDummyProvider("d", taijicfg.ProviderConfig{Type: "dummy", Model: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{
			{Role: RoleSystem, Content: "sys"},
			{Role: RoleUser, Content: "Ping"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "Pong" {
		t.Fatalf("content=%q", resp.Content)
	}
}

func TestDummyProviderIDAndUnknownPreset(t *testing.T) {
	p, err := newDummyProvider("my-dummy", taijicfg.ProviderConfig{Type: "dummy", Model: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	if p.ID() != "my-dummy" {
		t.Fatalf("id=%q", p.ID())
	}
	if _, err := newDummyProvider("bad", taijicfg.ProviderConfig{Type: "dummy", Model: "gamma"}); err == nil {
		t.Fatal("expected unknown preset error")
	}
}

func TestNewRegistryWithDummyProviders(t *testing.T) {
	cfg := &taijicfg.Config{
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
			"dummy_b": {Type: "dummy", Model: "beta"},
		},
	}
	reg, err := NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	a, err := reg.Get("dummy_a")
	if err != nil {
		t.Fatal(err)
	}
	b, err := reg.Get("dummy_b")
	if err != nil {
		t.Fatal(err)
	}
	if a.Type() != "dummy" || b.Type() != "dummy" {
		t.Fatalf("types: %s %s", a.Type(), b.Type())
	}
}
