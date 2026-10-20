package builtin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestWebFetchHTML(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<html><body><nav>skip</nav><h1>Title</h1><p>Hello <a href="/x">link</a></p><script>bad()</script></body></html>`))
	}))
	defer srv.Close()

	on := true
	toolsSettings = taijicfg.ToolsConfig{AllowLocalFetch: &on}
	out, err := (WebFetch{}).Run(context.Background(), map[string]any{
		"url": srv.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out, "bad()") || strings.Contains(out, "skip") {
		t.Fatalf("unexpected content: %s", out)
	}
	if !strings.Contains(out, "Title") || !strings.Contains(out, "Hello") {
		t.Fatalf("missing content: %s", out)
	}
}

func TestWebFetchBlocksLocalhost(t *testing.T) {
	toolsSettings = taijicfg.DefaultToolsConfig()
	_, err := (WebFetch{}).Run(context.Background(), map[string]any{
		"url": "http://127.0.0.1:1/",
	})
	if err == nil || !strings.Contains(err.Error(), "blocked") {
		t.Fatalf("expected block, got %v", err)
	}
}

func TestValidateFetchURLScheme(t *testing.T) {
	_, err := validateFetchURL("file:///etc/passwd", false)
	if err == nil || !strings.Contains(err.Error(), "http") {
		t.Fatalf("expected scheme error, got %v", err)
	}
}
