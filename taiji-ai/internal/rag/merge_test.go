package rag

import "testing"

func TestMergeHitsDedupesAndTopK(t *testing.T) {
	hits := []Hit{
		{Scope: "local", ID: "a", Score: 0.9, Text: "one"},
		{Scope: "global", ID: "b", Score: 0.95, Text: "two"},
		{Scope: "local", ID: "a", Score: 0.9, Text: "dup"},
	}
	out := mergeHits(hits, 2)
	if len(out) != 2 {
		t.Fatalf("len=%d", len(out))
	}
	if out[0].ID != "b" {
		t.Fatalf("expected highest score first, got %s", out[0].ID)
	}
}
