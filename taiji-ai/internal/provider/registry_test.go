package provider

import (
	"context"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestRegistryUnknownType(t *testing.T) {
	cfg := &taijicfg.Config{
		Providers: map[string]taijicfg.ProviderConfig{
			"x": {Type: "unknown"},
		},
	}
	_, err := NewRegistry(context.Background(), cfg, false)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRegistryOpenAIRequiresKey(t *testing.T) {
	cfg := &taijicfg.Config{
		Providers: map[string]taijicfg.ProviderConfig{
			"x": {Type: "openai", Model: "gpt-4o"},
		},
	}
	_, err := NewRegistry(context.Background(), cfg, false)
	if err == nil {
		t.Fatal("expected error")
	}
}
