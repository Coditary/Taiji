package ui

import (
	"strings"
	"testing"
)

func TestThemeTemplateDefaultName(t *testing.T) {
	out := ThemeTemplate("")
	if !strings.Contains(out, "custom") || !strings.Contains(out, "@theme {") {
		t.Fatalf("unexpected template: %q", out[:80])
	}
}

func TestThemeTemplateUsesName(t *testing.T) {
	out := ThemeTemplate("nord-custom")
	if !strings.Contains(out, "nord-custom") {
		t.Fatalf("missing theme id in template")
	}
}
