package agent

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestFilterMemoryToolDefs(t *testing.T) {
	all := []provider.ToolDef{
		{Name: "grep"},
		{Name: "memory_recall"},
		{Name: "memory_remember"},
		{Name: "list_dir"},
	}
	out := filterMemoryToolDefs(all)
	if len(out) != 2 || out[0].Name != "memory_recall" || out[1].Name != "memory_remember" {
		t.Fatalf("out=%v", out)
	}
}

func TestMemoryTrackerNeedsReflection(t *testing.T) {
	tr := newMemoryRunTracker()
	if tr.needsReflection() {
		t.Fatal("no errors yet")
	}
	tr.recordToolCall("grep", "{}", true, false)
	if !tr.needsReflection() {
		t.Fatal("expected reflection after tool error")
	}
	tr.rememberCalls = 1
	if tr.needsReflection() {
		t.Fatal("stored memory clears reflection need")
	}
}
