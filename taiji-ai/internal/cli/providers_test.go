package cli

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestPrintProviderTable(t *testing.T) {
	cfg := &taijicfg.Config{
		DefaultProvider: "main",
		Providers: map[string]taijicfg.ProviderConfig{
			"main": {Type: "dummy", Model: "alpha"},
			"alt":  {Type: "openai", Model: "gpt-test"},
		},
	}
	if err := printProviderTable(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestProviderFlags(t *testing.T) {
	pc := taijicfg.ProviderConfig{Pinned: true, Synced: true}
	if providerFlags(pc, true) != "default,pinned,synced" {
		t.Fatalf("flags=%q", providerFlags(pc, true))
	}
	if providerFlags(taijicfg.ProviderConfig{}, false) != "-" {
		t.Fatalf("expected dash for no flags")
	}
}
