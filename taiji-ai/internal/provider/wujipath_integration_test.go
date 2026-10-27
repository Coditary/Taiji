package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestGetProviderModelDetailsValidation(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir()}
	_, err := GetProviderModelDetails(context.Background(), cfg, "openai", "")
	if err == nil || !strings.Contains(err.Error(), "model id") {
		t.Fatalf("err=%v", err)
	}
	_, err = GetProviderModelDetails(context.Background(), nil, "openai", "gpt-4o")
	if err == nil || !strings.Contains(err.Error(), "config is nil") {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveTaijiWujiEndpointIntegration(t *testing.T) {
	root := t.TempDir()
	ep := ResolveTaijiWujiEndpoint(root, "auto", filepath.Join("..", "wuji-ai"))
	if ep == "" || !strings.HasPrefix(ep, "unix://") {
		t.Fatalf("endpoint = %q", ep)
	}
	if got := ResolveWujiRoot(&taijicfg.Config{Root: root, Wuji: taijicfg.WujiConfig{WujiRoot: "../wuji-ai"}}); got == "" {
		t.Fatal("expected wuji root")
	}
}

func TestWujiProviderEnsureClientInvalidEndpoint(t *testing.T) {
	root := t.TempDir()
	badWujiRoot := filepath.Join(root, "no-wuji-core")
	t.Setenv("WUJI_ROOT", badWujiRoot)
	cfg := &taijicfg.Config{
		Root: root,
		Wuji: taijicfg.WujiConfig{WujiRoot: badWujiRoot},
	}
	// Non-unix endpoint: skip auto-starting wuji-core; closed TCP port fails ping.
	p, err := newWujiProvider("wuji", taijicfg.ProviderConfig{
		Type:     "wuji",
		Endpoint: "127.0.0.1:1",
		Model:    "test",
		WujiRoot: badWujiRoot,
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	wp := p.(*wujiProvider)
	err = wp.ensureClient(context.Background())
	if err == nil {
		if wp.client != nil {
			_ = wp.client.Close()
		}
		t.Fatal("expected connect error")
	}
	if !strings.Contains(err.Error(), "connect wuji core") {
		t.Fatalf("err=%v", err)
	}
}

func TestGetProviderModelDetailsCatalogFallback(t *testing.T) {
	wujiRoot, err := filepath.Abs(filepath.Join("..", "wuji-ai"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(wujiRoot); err != nil {
		t.Skip("wuji-ai checkout not beside taiji-ai")
	}
	cfg := &taijicfg.Config{
		Root: t.TempDir(),
		Wuji: taijicfg.WujiConfig{WujiRoot: wujiRoot, Endpoint: "unix:///tmp/no-such-wuji.sock"},
	}
	t.Setenv("WUJI_ROOT", wujiRoot)
	_, err = GetProviderModelDetails(context.Background(), cfg, "openai", "gpt-4o")
	if err == nil {
		return
	}
	if !strings.Contains(err.Error(), "catalog") {
		t.Fatalf("expected catalog fallback error, got %v", err)
	}
}
