package stt

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestResolveDriverFromConfig(t *testing.T) {
	cfg := &taijicfg.Config{
		STT: taijicfg.STTConfig{Driver: "llama"},
	}
	if got := ResolveDriver(cfg); got != "llama" {
		t.Fatalf("got %q", got)
	}
	if ResolveDriver(nil) != "" {
		t.Fatal("nil cfg")
	}
}
