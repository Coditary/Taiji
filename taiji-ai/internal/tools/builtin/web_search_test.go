package builtin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestWebSearchRequiresQuery(t *testing.T) {
	_, err := (WebSearch{}).Run(context.Background(), map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "query") {
		t.Fatalf("err=%v", err)
	}
}

func TestBraveSearchViaHTTPTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") != "test-key" {
			http.Error(w, "bad key", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"web": map[string]any{
				"results": []map[string]string{
					{"title": "Taiji", "url": "https://example.com", "description": "agent runtime"},
				},
			},
		})
	}))
	defer srv.Close()

	prevBrave := braveWebSearchURL
	braveWebSearchURL = srv.URL
	defer func() { braveWebSearchURL = prevBrave }()

	out, err := braveSearch(context.Background(), "taiji", 5, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Taiji") || !strings.Contains(out, "example.com") {
		t.Fatalf("out=%q", out)
	}
}

func TestWebSearchUsesBraveWhenConfigured(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"web": map[string]any{
				"results": []map[string]string{
					{"title": "From Brave", "url": "https://brave.test", "description": "ok"},
				},
			},
		})
	}))
	defer srv.Close()

	prevBrave := braveWebSearchURL
	braveWebSearchURL = srv.URL
	defer func() { braveWebSearchURL = prevBrave }()

	toolsSettings = taijicfg.ToolsConfig{BraveSearchAPIKey: "secret"}
	out, err := (WebSearch{}).Run(context.Background(), map[string]any{"query": "x"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "From Brave") {
		t.Fatalf("out=%q", out)
	}
}

func TestDuckDuckGoSearchViaHTTPTest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"Abstract":       "A summary",
			"AbstractURL":    "https://ddg.test",
			"AbstractSource": "DDG",
		})
	}))
	defer srv.Close()

	prevDDG := duckDuckGoInstantURL
	duckDuckGoInstantURL = srv.URL + "?"
	defer func() { duckDuckGoInstantURL = prevDDG }()

	out, err := duckDuckGoSearch(context.Background(), "query", 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "A summary") {
		t.Fatalf("out=%q", out)
	}
}
