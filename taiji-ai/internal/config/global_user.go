package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// applyGlobalUserPrefs overlays UI/STT from the global config onto cfg.
func applyGlobalUserPrefs(cfg *Config) {
	if cfg == nil {
		return
	}
	global := loadGlobalFileConfig()
	if global == nil {
		return
	}
	overlayUI(&cfg.UI, global.UI)
	mergeSTT(&cfg.STT, global.STT)
}

// SaveGlobalUserPrefs persists appearance and STT preferences to the global config file.
func SaveGlobalUserPrefs(ui UIConfig, stt STTConfig) error {
	dir := GlobalConfigDir()
	if dir == "" {
		return fmt.Errorf("global config dir is not set")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create global config dir: %w", err)
	}
	path := filepath.Join(dir, configFile)
	var fc fileConfig
	if raw, err := os.ReadFile(path); err == nil {
		_ = yaml.Unmarshal(raw, &fc)
	}
	fc.UI = ui
	fc.STT = stt
	raw, err := yaml.Marshal(fc)
	if err != nil {
		return fmt.Errorf("marshal global config: %w", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		return fmt.Errorf("write global config: %w", err)
	}
	return nil
}

func overlayUI(dst *UIConfig, src UIConfig) {
	if dst == nil {
		return
	}
	if strings.TrimSpace(src.Theme) != "" {
		dst.Theme = strings.TrimSpace(src.Theme)
	}
	if strings.TrimSpace(src.Locale) != "" {
		dst.Locale = strings.TrimSpace(src.Locale)
	}
	if strings.TrimSpace(src.CornerStyle) != "" {
		dst.CornerStyle = strings.TrimSpace(src.CornerStyle)
	}
	if strings.TrimSpace(src.ToolCallDensity) != "" {
		dst.ToolCallDensity = strings.TrimSpace(src.ToolCallDensity)
	}
	if strings.TrimSpace(src.FileDiffLayout) != "" {
		dst.FileDiffLayout = strings.TrimSpace(src.FileDiffLayout)
	}
	if src.ActivitySpacingLines != 0 {
		dst.ActivitySpacingLines = src.ActivitySpacingLines
	}
	if src.MessageSpacingLines != 0 {
		dst.MessageSpacingLines = src.MessageSpacingLines
	}
	if src.ActivityShowPipelineTitle != nil {
		dst.ActivityShowPipelineTitle = src.ActivityShowPipelineTitle
	}
	if src.ToolExpandShowParameters != nil {
		dst.ToolExpandShowParameters = src.ToolExpandShowParameters
	}
	if src.ToolExpandShowResult != nil {
		dst.ToolExpandShowResult = src.ToolExpandShowResult
	}
	if strings.TrimSpace(src.ToolExpandArgsFormat) != "" {
		dst.ToolExpandArgsFormat = strings.TrimSpace(src.ToolExpandArgsFormat)
	}
	if strings.TrimSpace(src.ToolExpandResultFormat) != "" {
		dst.ToolExpandResultFormat = strings.TrimSpace(src.ToolExpandResultFormat)
	}
	if src.CodeBlockWrap != nil {
		dst.CodeBlockWrap = src.CodeBlockWrap
	}
	if src.ChatMaxWidth != 0 {
		dst.ChatMaxWidth = src.ChatMaxWidth
	}
	if src.ReduceMotion != nil {
		dst.ReduceMotion = src.ReduceMotion
	}
	if src.MessageTimestamps != nil {
		dst.MessageTimestamps = src.MessageTimestamps
	}
	if strings.TrimSpace(src.MessageTimeClock) != "" {
		dst.MessageTimeClock = strings.TrimSpace(src.MessageTimeClock)
	}
	if src.StickyUserMessage != nil {
		dst.StickyUserMessage = src.StickyUserMessage
	}
	if src.MessageFillBackground != nil {
		dst.MessageFillBackground = src.MessageFillBackground
	}
	if src.UserMessageFillBackground != nil {
		dst.UserMessageFillBackground = src.UserMessageFillBackground
	}
	if src.AssistantMessageFillBackground != nil {
		dst.AssistantMessageFillBackground = src.AssistantMessageFillBackground
	}
	if strings.TrimSpace(src.UserMessageStyle) != "" {
		dst.UserMessageStyle = strings.TrimSpace(src.UserMessageStyle)
	}
	if strings.TrimSpace(src.UserMessageAlign) != "" {
		dst.UserMessageAlign = strings.TrimSpace(src.UserMessageAlign)
	}
	if src.UserMessageMaxWidthPct != 0 {
		dst.UserMessageMaxWidthPct = src.UserMessageMaxWidthPct
	}
	if strings.TrimSpace(src.AssistantMessageStyle) != "" {
		dst.AssistantMessageStyle = strings.TrimSpace(src.AssistantMessageStyle)
	}
	if strings.TrimSpace(src.AssistantMessageAlign) != "" {
		dst.AssistantMessageAlign = strings.TrimSpace(src.AssistantMessageAlign)
	}
	if src.AssistantMessageMaxWidthPct != 0 {
		dst.AssistantMessageMaxWidthPct = src.AssistantMessageMaxWidthPct
	}
	if src.AutoCompactContextPct != 0 {
		dst.AutoCompactContextPct = src.AutoCompactContextPct
	}
	if strings.TrimSpace(src.KeybindingStyle) != "" {
		dst.KeybindingStyle = strings.TrimSpace(src.KeybindingStyle)
	}
	if strings.TrimSpace(src.VimEscapeSequence) != "" {
		dst.VimEscapeSequence = strings.TrimSpace(src.VimEscapeSequence)
	}
	if strings.TrimSpace(src.LeaderKey) != "" {
		dst.LeaderKey = strings.TrimSpace(src.LeaderKey)
	}
	if strings.TrimSpace(src.Keybindings.OpenThemePicker) != "" {
		dst.Keybindings.OpenThemePicker = strings.TrimSpace(src.Keybindings.OpenThemePicker)
	}
	overlayTurnMetrics(&dst.TurnMetrics, src.TurnMetrics)
	overlayToolLabels(&dst.ToolLabels, src.ToolLabels)
}

func overlayTurnMetrics(dst *TurnMetricsUIConfig, src TurnMetricsUIConfig) {
	if dst == nil {
		return
	}
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if src.ShowActions != nil {
		dst.ShowActions = src.ShowActions
	}
	if src.Compact != nil {
		dst.Compact = src.Compact
	}
	if strings.TrimSpace(src.Separator) != "" {
		dst.Separator = strings.TrimSpace(src.Separator)
	}
	if src.MaxFieldsPerLine != 0 {
		dst.MaxFieldsPerLine = src.MaxFieldsPerLine
	}
	if len(src.Fields) > 0 {
		dst.Fields = append([]string(nil), src.Fields...)
	}
}

func overlayToolLabels(dst *ToolLabelsUIConfig, src ToolLabelsUIConfig) {
	if dst == nil {
		return
	}
	if strings.TrimSpace(src.Locale) != "" {
		dst.Locale = strings.TrimSpace(src.Locale)
	}
}
