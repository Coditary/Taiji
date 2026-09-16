package agent

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestResponseRequestsContinue(t *testing.T) {
	trigger := "<|continue|>"
	resp := &provider.CompletionResponse{Content: "Schritt 1 done\n<|continue|>"}
	if !responseRequestsContinue(resp, trigger) {
		t.Fatal("expected continue")
	}
	resp2 := &provider.CompletionResponse{Content: "Fertig."}
	if responseRequestsContinue(resp2, trigger) {
		t.Fatal("expected no continue")
	}
}

func TestStripContinueTrigger(t *testing.T) {
	trigger := "<|continue|>"
	got := stripContinueTrigger("a\n<|continue|>\n", trigger)
	if got != "a" {
		t.Fatalf("got %q", got)
	}
}
