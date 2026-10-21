package uilocale

import "testing"

func TestLookupJapaneseToolName(t *testing.T) {
	if got := Lookup("ja", "tools", "names.grep"); got != "検索" {
		t.Fatalf("got %q want 検索", got)
	}
}

func TestLookupPlaceholderFallsBackToEnglish(t *testing.T) {
	// metrics.total is "..." in some locales until translated; runtime uses English.
	if got := Lookup("fr", "settings", "categories.agent_short"); got != "Agent" {
		t.Fatalf("got %q want English fallback for placeholder", got)
	}
}

func TestLookupMetricsFallbackEnglish(t *testing.T) {
	if got := Lookup("it", "tools", "metrics.nonexistent_key"); got != "" {
		t.Fatalf("got %q", got)
	}
	if got := Lookup("xx", "tools", "metrics.total"); got != "total" {
		t.Fatalf("got %q want English fallback", got)
	}
}

func TestPrefixMapToolNames(t *testing.T) {
	m := PrefixMap("de", "tools", "names.")
	if m["read_file"] != "Lesen" {
		t.Fatalf("got %q", m["read_file"])
	}
}
