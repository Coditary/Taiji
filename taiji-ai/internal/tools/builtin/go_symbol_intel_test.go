package builtin

import (
	"strings"
	"testing"
)

func TestGoSymbolIntelDefinition(t *testing.T) {
	dir := "."
	out, err := (GoSymbolIntel{}).Run(t.Context(), map[string]any{
		"action": "definition",
		"symbol": "ReadFile",
		"dir":    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ReadFile") {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestGoSymbolIntelPkgSummary(t *testing.T) {
	dir := "."
	out, err := (GoSymbolIntel{}).Run(t.Context(), map[string]any{
		"action": "pkg_summary",
		"dir":    dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "package builtin") {
		t.Fatalf("unexpected: %q", out)
	}
}
