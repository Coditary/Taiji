package config

import "testing"

func TestToolDisplayTitleUsesLocalePreset(t *testing.T) {
	cfg := &Config{}
	cfg.UI.ToolLabels.Locale = "en"
	if got := cfg.ToolDisplayTitle("read_file"); got != "Read" {
		t.Fatalf("got %q want Read", got)
	}
	if got := cfg.ToolDisplayTitle("list_dir"); got != "List" {
		t.Fatalf("got %q want List", got)
	}
}

func TestToolDisplayTitleOverride(t *testing.T) {
	cfg := &Config{}
	cfg.UI.ToolLabels.Locale = "en"
	cfg.UI.ToolLabels.Tools = map[string]string{
		"read_file": "Open file",
	}
	if got := cfg.ToolDisplayTitle("read_file"); got != "Open file" {
		t.Fatalf("got %q", got)
	}
}

func TestToolDisplayTitleFallbackEmpty(t *testing.T) {
	cfg := &Config{}
	cfg.UI.ToolLabels.Locale = ""
	cfg.UI.ToolLabels.Tools = map[string]string{}
	if got := cfg.ToolDisplayTitle("custom_tool"); got != "" {
		t.Fatalf("expected empty fallback, got %q", got)
	}
}

func TestGermanToolLabels(t *testing.T) {
	cfg := &Config{}
	cfg.UI.ToolLabels.Locale = "de"
	if got := cfg.ToolDisplayTitle("read_file"); got != "Lesen" {
		t.Fatalf("got %q want Lesen", got)
	}
}
