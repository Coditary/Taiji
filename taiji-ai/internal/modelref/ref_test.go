package modelref

import "testing"

func TestDisplay(t *testing.T) {
	if got := Display("ollama/gemma4_26b-128k-nothink.gguf"); got != "gemma4_26b-128k-nothink" {
		t.Fatalf("Display: %q", got)
	}
	if got := Display("claude-sonnet-4"); got != "claude-sonnet-4" {
		t.Fatalf("Display cloud: %q", got)
	}
}

func TestExpand(t *testing.T) {
	candidates := []string{"ollama/gemma4_26b-128k-nothink.gguf"}
	if got := Expand("gemma4_26b-128k-nothink", candidates); got != candidates[0] {
		t.Fatalf("Expand: %q", got)
	}
	if got := Expand("ollama/gemma4_26b-128k-nothink.gguf", candidates); got != candidates[0] {
		t.Fatalf("Expand canonical: %q", got)
	}
}
