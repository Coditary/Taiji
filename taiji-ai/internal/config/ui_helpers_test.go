package config

import "testing"

func TestNormalizeUIShortcut(t *testing.T) {
	if got := NormalizeUIShortcut("ctrl+t"); got != "Ctrl+T" {
		t.Fatalf("got %q", got)
	}
	if NormalizeUIShortcut("") != "" {
		t.Fatal("empty")
	}
	if got := NormalizeUIShortcut("meta+shift+p"); got != "Meta+Shift+P" {
		t.Fatalf("got %q", got)
	}
}

func TestNormalizeUserMessageStyleAndAlign(t *testing.T) {
	cases := map[string]string{
		"plain": "plain", "bubble": "messenger", "": "messenger", "weird": "messenger",
	}
	for in, want := range cases {
		if got := NormalizeUserMessageStyle(in); got != want {
			t.Fatalf("%q: got %q", in, got)
		}
	}
	if NormalizeMessageAlign("right") != "right" {
		t.Fatal("right")
	}
	if NormalizeMessageAlign("unknown") != "auto" {
		t.Fatal("auto default")
	}
}

func TestUIConfigDefaultsAndClamps(t *testing.T) {
	def := DefaultUIConfig()
	if !def.TurnMetrics.EnabledOrDefault() {
		t.Fatal("metrics enabled by default")
	}
	if def.UserMessageStyleOrDefault() != "messenger" {
		t.Fatal("user style")
	}
	if def.AssistantMessageStyleOrDefault() != "messenger" {
		t.Fatal("assistant style")
	}
	if def.MessageTimeClockOrDefault() != "24h" {
		t.Fatal("clock")
	}
	if def.UserMessageMaxWidthPctOrDefault() != 80 {
		t.Fatal("width default")
	}
	c := UIConfig{UserMessageMaxWidthPct: 10, AssistantMessageMaxWidthPct: 200}
	if c.UserMessageMaxWidthPctOrDefault() != 40 {
		t.Fatal("clamp low")
	}
	if c.AssistantMessageMaxWidthPctOrDefault() != 100 {
		t.Fatal("clamp high")
	}
	if !(UIConfig{CornerStyle: "rounded"}).CornersRoundedOrDefault() {
		t.Fatal("rounded")
	}
}

func TestNormalizeUILocale(t *testing.T) {
	if NormalizeUILocale("de") != "de" {
		t.Fatal("de")
	}
	if NormalizeUILocale("xx") != "en" {
		t.Fatal("fallback en")
	}
	if (UIConfig{Locale: "es"}).LocaleOrDefault() != "es" {
		t.Fatal("locale or default")
	}
}

func TestTurnMetricsPresets(t *testing.T) {
	dev := TurnMetricsUIConfig{Preset: "dev"}
	fields := dev.ResolvedFields()
	if len(fields) < 5 {
		t.Fatalf("dev preset fields=%v", fields)
	}
	minimal := TurnMetricsUIConfig{Preset: "minimal"}
	if len(minimal.ResolvedFields()) != 1 {
		t.Fatal("minimal")
	}
	custom := TurnMetricsUIConfig{Fields: []string{"total", "bogus", "cost", "total"}}
	got := custom.ResolvedFields()
	if len(got) != 2 || got[0] != TurnMetricFieldTotal || got[1] != TurnMetricFieldCost {
		t.Fatalf("normalized fields=%v", got)
	}
	if custom.SeparatorOrDefault() != " · " {
		t.Fatal("default separator")
	}
	if (TurnMetricsUIConfig{Separator: "|"}).SeparatorOrDefault() != "|" {
		t.Fatal("custom separator")
	}
}

func TestUINormalizeHelpers(t *testing.T) {
	if NormalizeToolCallDensity("compact") != "clustered" {
		t.Fatal("compact alias")
	}
	if NormalizeToolCallDensity("weird") != "subtree" {
		t.Fatal("default density")
	}
	if NormalizeFileDiffLayout("split") != "side_by_side" {
		t.Fatal("split layout")
	}
	if ClampSpacingLines(-1) != 0 || ClampSpacingLines(9) != 4 {
		t.Fatal("clamp spacing")
	}
	if NormalizeToolExpandArgsFormat("json") != "json" {
		t.Fatal("args json")
	}
	if NormalizeToolExpandResultFormat("JSON") != "json" {
		t.Fatal("result json")
	}
}

func TestUIKeybindingDefaults(t *testing.T) {
	c := UIConfig{}
	if c.KeybindingsOrDefault().OpenThemePicker != "Ctrl+T" {
		t.Fatal("theme picker default")
	}
	if c.KeybindingStyleOrDefault() != "standard" {
		t.Fatal("standard keybinding style")
	}
	if (UIConfig{KeybindingStyle: "vim"}).KeybindingStyleOrDefault() != "vim" {
		t.Fatal("vim style")
	}
	if c.VimEscapeSequenceOrDefault() != "jk" {
		t.Fatal("vim escape")
	}
	if c.LeaderKeyOrDefault() != "Space" {
		t.Fatal("leader key")
	}
	if !c.MessageTimestampsOrDefault() || !c.StickyUserMessageOrDefault() {
		t.Fatal("message chrome defaults")
	}
}

func TestUIConfigOptionalDefaults(t *testing.T) {
	c := UIConfig{
		ToolCallDensity:      "inline",
		FileDiffLayout:       "side_by_side",
		ActivitySpacingLines: 2,
		MessageSpacingLines:  3,
		ToolExpandArgsFormat: "json",
		ToolExpandResultFormat: "json",
	}
	if c.ToolCallDensityOrDefault() != "inline" {
		t.Fatal("density")
	}
	if c.FileDiffLayoutOrDefault() != "side_by_side" {
		t.Fatal("layout")
	}
	if c.ActivitySpacingLinesOrDefault() != 2 {
		t.Fatal("activity spacing")
	}
	if c.MessageSpacingLinesOrDefault() != 3 {
		t.Fatal("message spacing")
	}
	if !c.ActivityShowPipelineTitleOrDefault() {
		t.Fatal("pipeline title default true")
	}
	if !c.ToolExpandShowParametersOrDefault() || !c.ToolExpandShowResultOrDefault() {
		t.Fatal("expand defaults")
	}
	if c.ToolExpandArgsFormatOrDefault() != "json" || c.ToolExpandResultFormatOrDefault() != "json" {
		t.Fatal("expand formats")
	}
	compactTrue := true
	if (UIConfig{TurnMetrics: TurnMetricsUIConfig{Compact: &compactTrue}}).ToolCallDensityOrDefault() != "clustered" {
		t.Fatal("legacy compact -> clustered")
	}
}
