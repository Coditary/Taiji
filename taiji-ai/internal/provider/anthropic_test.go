package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestAnthropicProviderComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"content": []map[string]any{{"type": "text", "text": "hello anthropic"}},
		})
	}))
	defer srv.Close()

	t.Setenv("ANTHROPIC_TEST_KEY", "sk-test")
	p, err := newAnthropicProvider("ant", taijicfg.ProviderConfig{
		Type:       "anthropic",
		BaseURL:    srv.URL,
		APIKeyEnv:  "ANTHROPIC_TEST_KEY",
		Model:      "claude-test",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "ping"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "hello anthropic" {
		t.Fatalf("content=%q", resp.Content)
	}
	if p.ID() != "ant" || p.Type() != "anthropic" {
		t.Fatalf("id/type=%s %s", p.ID(), p.Type())
	}
}

func TestAnthropicProviderAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "overloaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	t.Setenv("ANTHROPIC_TEST_KEY", "sk-test")
	p, err := newAnthropicProvider("ant", taijicfg.ProviderConfig{
		Type: "anthropic", BaseURL: srv.URL, APIKeyEnv: "ANTHROPIC_TEST_KEY", Model: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "x"}},
	})
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("expected api error, got %v", err)
	}
}
