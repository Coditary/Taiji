package provider

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestWujiProviderUsesProviderID(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/tmp"}
	p, err := newWujiProvider("echo", taijicfg.ProviderConfig{
		Type:       "wuji",
		ProviderID: "echo",
		Model:      "test-model",
	}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	wp, ok := p.(*wujiProvider)
	if !ok {
		t.Fatal("expected wujiProvider")
	}
	if wp.driverID != "echo" {
		t.Fatalf("driverID: %q", wp.driverID)
	}
}

func TestProviderConfigChanged(t *testing.T) {
	before := taijicfg.ProviderConfig{Type: "wuji", ProviderID: "vllm", Model: "a"}
	after := taijicfg.ProviderConfig{Type: "wuji", ProviderID: "vllm", Model: "b"}
	if !providerConfigChanged(before, after) {
		t.Fatal("expected model change to be detected")
	}
}
