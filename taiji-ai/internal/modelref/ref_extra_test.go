package modelref

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestCandidatesAndExpandSynthetic(t *testing.T) {
	cfg := &taijicfg.Config{
		Providers: map[string]taijicfg.ProviderConfig{
			"p1": {Model: "ollama/foo.gguf"},
		},
	}
	got := Candidates(cfg, "p1", "extra/model")
	if len(got) != 2 || got[0] != "extra/model" {
		t.Fatalf("candidates=%v", got)
	}
	if Candidates(nil, "p1") != nil {
		t.Fatal("nil cfg")
	}
	if Expand("", nil) != "" {
		t.Fatal("empty expand")
	}
	if Expand("foo", nil) != "ollama/foo.gguf" {
		t.Fatalf("synthetic expand")
	}
}

func TestDisplayEdgeCases(t *testing.T) {
	if Display("") != "" {
		t.Fatal("empty")
	}
	if Display("plain-model") != "plain-model" {
		t.Fatal("unchanged cloud")
	}
}
