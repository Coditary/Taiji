package agent

import (
	"testing"
)

func TestParseEmbeddedToolCalls_single(t *testing.T) {
	content := `[tool call] read_file {"path": "path"}`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Fatalf("unexpected name: %s", calls[0].Name)
	}
	if calls[0].Args != `{"path": "path"}` {
		t.Fatalf("unexpected args: %s", calls[0].Args)
	}
}

func TestParseEmbeddedToolCalls_withPreamble(t *testing.T) {
	content := "Ich schaue kurz nach.\n\n[tool call] read_file {\"path\": \"Makefile\"}"
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Name != "read_file" {
		t.Fatalf("unexpected name: %s", calls[0].Name)
	}
}

func TestParseEmbeddedToolCalls_multipleAtEnd(t *testing.T) {
	content := `[tool call] list_dir {"path": "."}
[tool call] read_file {"path": "go.mod"}`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(calls))
	}
	if calls[0].Name != "list_dir" || calls[1].Name != "read_file" {
		t.Fatalf("unexpected names: %s, %s", calls[0].Name, calls[1].Name)
	}
}

func TestParseEmbeddedToolCalls_parsesTrailingAnswerAfterCall(t *testing.T) {
	content := "Ich schaue kurz nach.\n[tool call] read_file {\"path\": \"Makefile\"}\nHier ist die Antwort."
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call when premature answer follows tool call, got %+v", calls)
	}
	stripped := StripEmbeddedToolCalls(content)
	if stripped != "Ich schaue kurz nach." {
		t.Fatalf("unexpected stripped content: %q", stripped)
	}
}

func TestParseEmbeddedToolCalls_parsesGluedAnswerAfterJSON(t *testing.T) {
	content := `[tool call] invoke_skill {"names": ["hallo"]}Alles klar, Skills geladen.`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 || calls[0].Name != "invoke_skill" {
		t.Fatalf("unexpected calls: %+v", calls)
	}
	if StripEmbeddedToolCalls(content) != "" {
		t.Fatalf("expected empty preamble")
	}
}

func TestParseEmbeddedToolCalls_onlyTrailingBlock(t *testing.T) {
	content := `[tool call] list_dir {"path": "."}
Zwischen dem ersten und zweiten Call.
[tool call] read_file {"path": "go.mod"}`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 || calls[0].Name != "read_file" {
		t.Fatalf("expected only trailing tool call, got %+v", calls)
	}
}

func TestStripEmbeddedToolCalls(t *testing.T) {
	content := "Ich schaue kurz nach.\n\n[tool call] read_file {\"path\": \"Makefile\"}"
	stripped := StripEmbeddedToolCalls(content)
	if stripped != "Ich schaue kurz nach." {
		t.Fatalf("unexpected stripped content: %q", stripped)
	}
}

func TestParseEmbeddedToolCalls_noMarker(t *testing.T) {
	content := "Nur normaler Text ohne Tools."
	if calls := ParseEmbeddedToolCalls(content); len(calls) != 0 {
		t.Fatalf("expected no calls, got %+v", calls)
	}
}

func TestParseEmbeddedToolCalls_multipleInvokeSkill(t *testing.T) {
	content := `Ich lade die Skills.
[tool call] invoke_skill {"name": "hallo"}
[tool call] invoke_skill {"name": "review"}
[tool call] invoke_skill {"names": ["automate"]}`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 3 {
		t.Fatalf("expected 3 calls, got %d: %+v", len(calls), calls)
	}
	if calls[0].Name != "invoke_skill" || calls[2].Name != "invoke_skill" {
		t.Fatalf("unexpected tool names: %+v", calls)
	}
}

func TestParseEmbeddedToolCalls_batchInvokeSkillNames(t *testing.T) {
	content := `[tool call] invoke_skill {"names": ["hallo", "review", "automate"]}`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}
	if calls[0].Args != `{"names": ["hallo", "review", "automate"]}` {
		t.Fatalf("unexpected args: %s", calls[0].Args)
	}
}

func TestStreamPreambleBeforeToolCalls_prefersStreamed(t *testing.T) {
	streamed := "Ich lade drei Skills.\n[tool call] invoke_skill {\"names\": [\"a\"]}"
	resp := `[tool call] invoke_skill {"names": ["a"]}`
	got := StreamPreambleBeforeToolCalls(streamed, resp)
	if got != "Ich lade drei Skills." {
		t.Fatalf("unexpected preamble: %q", got)
	}
}

func TestStreamPreambleBeforeToolCalls_markerFallback(t *testing.T) {
	streamed := "Kurz vorweg.\n[tool call] invoke_skill {\"name\": \"x\"}Antwort"
	got := StreamPreambleBeforeToolCalls(streamed, "")
	if got != "Kurz vorweg." {
		t.Fatalf("unexpected preamble: %q", got)
	}
}

func TestParseEmbeddedToolCalls_caseInsensitiveMarker(t *testing.T) {
	content := `[TOOL CALL] grep {"pattern": "func main"}`
	calls := ParseEmbeddedToolCalls(content)
	if len(calls) != 1 || calls[0].Name != "grep" {
		t.Fatalf("unexpected parse result: %+v", calls)
	}
}
