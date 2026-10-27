package agent

import "testing"

func TestMemoryRunTrackerRecallAndReflection(t *testing.T) {
	tr := newMemoryRunTracker()
	tr.recordToolCall("grep", `{}`, true, false)
	if !tr.needsReflection() {
		t.Fatal("expected reflection after non-memory tool error")
	}
	tr.recordToolCall("memory_remember", `{"content":"x"}`, false, false)
	if tr.needsReflection() {
		t.Fatal("remember should suppress reflection")
	}
	tr.recordToolCall("memory_recall", `{"kind":"fact"}`, false, true)
	if tr.autoRecalls != 1 || tr.recallByKind["fact"] != 1 {
		t.Fatalf("recall stats: auto=%d kinds=%v", tr.autoRecalls, tr.recallByKind)
	}
	tr.setRecalledID(" mem-1 ")
	if tr.lastRecalledID != "mem-1" {
		t.Fatalf("id=%q", tr.lastRecalledID)
	}
}

func TestParseStringArgJSON(t *testing.T) {
	if got := parseStringArg(`{"kind":"note"}`, "kind"); got != "note" {
		t.Fatalf("got %q", got)
	}
	if parseStringArg("not-json", "kind") != "" {
		t.Fatal("bad json")
	}
}
