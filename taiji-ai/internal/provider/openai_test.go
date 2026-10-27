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

func TestOpenAIProviderComplete(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{"content": "pong"},
			}},
		})
	}))
	defer srv.Close()

	p, err := newOpenAIProvider("oa", taijicfg.ProviderConfig{
		Type:    "openai",
		BaseURL: srv.URL,
		APIKey:  "test-key",
		Model:   "gpt-test",
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
	if resp.Content != "pong" {
		t.Fatalf("content=%q", resp.Content)
	}
}

func TestNewOpenAIProviderRequiresAPIKey(t *testing.T) {
	_, err := newOpenAIProvider("oa", taijicfg.ProviderConfig{Type: "openai", Model: "m"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestOpenAIProviderToolCallsAndAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{{
				"message": map[string]any{
					"content": "",
					"tool_calls": []map[string]any{{
						"id": "call_1",
						"function": map[string]any{
							"name":      "read_file",
							"arguments": `{"path":"x"}`,
						},
					}},
				},
				"finish_reason": "tool_calls",
			}},
			"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 2},
		})
	}))
	defer srv.Close()

	p, err := newOpenAIProvider("oa", taijicfg.ProviderConfig{
		Type: "openai", BaseURL: srv.URL, APIKey: "k", Model: "m",
	})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := p.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "go"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.ToolCalls) != 1 || resp.ToolCalls[0].Name != "read_file" {
		t.Fatalf("tool calls=%+v", resp.ToolCalls)
	}
	if p.ID() != "oa" || p.Type() != "openai" {
		t.Fatalf("id/type=%s %s", p.ID(), p.Type())
	}

	errSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer errSrv.Close()
	p2, _ := newOpenAIProvider("oa2", taijicfg.ProviderConfig{
		Type: "openai", BaseURL: errSrv.URL, APIKey: "k", Model: "m",
	})
	if _, err := p2.Complete(context.Background(), CompletionRequest{
		Messages: []Message{{Role: RoleUser, Content: "x"}},
	}); err == nil || !strings.Contains(err.Error(), "429") {
		t.Fatalf("expected api error, got %v", err)
	}
}

func TestParseOpenAIResponseEmptyChoices(t *testing.T) {
	if _, err := parseOpenAIResponse([]byte(`{"choices":[]}`)); err == nil {
		t.Fatal("expected empty choices error")
	}
}
