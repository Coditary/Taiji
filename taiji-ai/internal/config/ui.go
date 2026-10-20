package config

import "strings"

// UIConfig controls Taiji TUI presentation.
type UIConfig struct {
	TurnMetrics TurnMetricsUIConfig `yaml:"turn_metrics"`
	ToolLabels  ToolLabelsUIConfig  `yaml:"tool_labels"`
	// Theme is a taiji-tui stylesheet id (built-in: dark, high_contrast; or ~/.config/taiji/themes/<id>.theme).
	Theme string `yaml:"theme,omitempty"`
	// CornerStyle is "square", "rounded", or "flat" (fill only, no outline) for composer and user bubbles.
	CornerStyle string `yaml:"corner_style,omitempty"`
	// CodeBlockWrap enables soft-wrapping inside fenced code blocks.
	CodeBlockWrap *bool `yaml:"code_block_wrap,omitempty"`
	// ChatMaxWidth limits message column width in cells (0 = use full pane).
	ChatMaxWidth int `yaml:"chat_max_width,omitempty"`
	// ReduceMotion is reserved for future animations (stored preference).
	ReduceMotion *bool `yaml:"reduce_motion,omitempty"`
	// MessageTimestamps shows send time on user message bubbles in the TUI.
	MessageTimestamps *bool `yaml:"message_timestamps,omitempty"`
	// MessageTimeClock is "24h" or "12h" (AM/PM).
	MessageTimeClock string `yaml:"message_time_clock,omitempty"`
	// StickyUserMessage pins a preview of your prompt at the top while scrolling long replies.
	StickyUserMessage *bool `yaml:"sticky_user_message,omitempty"`
}

// TurnMetricsUIConfig selects which per-turn timing fields appear under assistant replies.
type TurnMetricsUIConfig struct {
	Enabled          *bool    `yaml:"enabled,omitempty"`
	ShowActions      *bool    `yaml:"show_actions,omitempty"`
	Preset           string   `yaml:"preset,omitempty"`
	Fields           []string `yaml:"fields,omitempty"`
	MaxFieldsPerLine int      `yaml:"max_fields_per_line,omitempty"`
	Separator        string   `yaml:"separator,omitempty"`
	Compact          *bool    `yaml:"compact,omitempty"`
	normalized       []string
}

// DefaultUIConfig returns TUI defaults.
func DefaultUIConfig() UIConfig {
	enabled := true
	showActions := true
	messageTimestamps := true
	stickyUserMessage := true
	return UIConfig{
		Theme:               "dark",
		MessageTimestamps:   &messageTimestamps,
		MessageTimeClock:    "24h",
		StickyUserMessage:   &stickyUserMessage,
		TurnMetrics: TurnMetricsUIConfig{
			Enabled:          &enabled,
			ShowActions:      &showActions,
			MaxFieldsPerLine: 4,
			Fields: []string{
				TurnMetricFieldTotal,
				TurnMetricFieldWait,
				TurnMetricFieldGeneration,
				TurnMetricFieldOutputTokens,
				TurnMetricFieldTokensPerSec,
			},
		},
		ToolLabels: ToolLabelsUIConfig{
			Locale: "en",
		},
	}
}

// MessageTimestampsOrDefault reports whether user message send times are shown.
func (c UIConfig) MessageTimestampsOrDefault() bool {
	if c.MessageTimestamps == nil {
		return true
	}
	return *c.MessageTimestamps
}

// StickyUserMessageOrDefault reports whether scrolled prompts stay pinned at the top of the chat.
func (c UIConfig) StickyUserMessageOrDefault() bool {
	if c.StickyUserMessage == nil {
		return true
	}
	return *c.StickyUserMessage
}

// CornersRoundedOrDefault reports whether framed UI chrome uses rounded box corners.
func (c UIConfig) CornersRoundedOrDefault() bool {
	return strings.EqualFold(strings.TrimSpace(c.CornerStyle), "rounded")
}

// MessageTimeClockOrDefault returns "24h" or "12h".
func (c UIConfig) MessageTimeClockOrDefault() string {
	switch strings.ToLower(strings.TrimSpace(c.MessageTimeClock)) {
	case "12h", "12", "am_pm", "ampm":
		return "12h"
	default:
		return "24h"
	}
}

const (
	TurnMetricFieldTotal        = "total"
	TurnMetricFieldWait         = "wait"
	TurnMetricFieldThought      = "thought"
	TurnMetricFieldGeneration   = "generation"
	TurnMetricFieldOutputTokens = "output_tokens"
	TurnMetricFieldInputTokens  = "input_tokens"
	TurnMetricFieldTokensPerSec = "tokens_per_sec"
	TurnMetricFieldCost         = "cost"
	TurnMetricFieldTools        = "tools"
	TurnMetricFieldToolTime     = "tool_time"
	TurnMetricFieldSteps        = "steps"
	TurnMetricFieldContext      = "context"
	TurnMetricFieldContextPct   = "context_pct"
)

var turnMetricsPresets = map[string][]string{
	"minimal": {
		TurnMetricFieldTotal,
	},
	"tokens": {
		TurnMetricFieldOutputTokens,
		TurnMetricFieldTokensPerSec,
	},
	"dev": {
		TurnMetricFieldTotal,
		TurnMetricFieldWait,
		TurnMetricFieldThought,
		TurnMetricFieldGeneration,
		TurnMetricFieldOutputTokens,
		TurnMetricFieldInputTokens,
		TurnMetricFieldTokensPerSec,
		TurnMetricFieldTools,
		TurnMetricFieldToolTime,
		TurnMetricFieldCost,
		TurnMetricFieldSteps,
	},
	"verbose": {
		TurnMetricFieldTotal,
		TurnMetricFieldWait,
		TurnMetricFieldThought,
		TurnMetricFieldGeneration,
		TurnMetricFieldOutputTokens,
		TurnMetricFieldInputTokens,
		TurnMetricFieldTokensPerSec,
		TurnMetricFieldTools,
		TurnMetricFieldToolTime,
		TurnMetricFieldCost,
		TurnMetricFieldSteps,
		TurnMetricFieldContext,
		TurnMetricFieldContextPct,
	},
}

// EnabledOrDefault reports whether turn metrics text should be shown.
func (c TurnMetricsUIConfig) EnabledOrDefault() bool {
	if c.Enabled == nil {
		return true
	}
	return *c.Enabled
}

// ShowActionsOrDefault reports whether redo/copy controls appear under finished turns.
func (c TurnMetricsUIConfig) ShowActionsOrDefault() bool {
	if c.ShowActions == nil {
		return true
	}
	return *c.ShowActions
}

// CompactOrDefault reports whether metric labels are shortened.
func (c TurnMetricsUIConfig) CompactOrDefault() bool {
	if c.Compact == nil {
		return false
	}
	return *c.Compact
}

// SeparatorOrDefault returns the field separator.
func (c TurnMetricsUIConfig) SeparatorOrDefault() string {
	sep := strings.TrimSpace(c.Separator)
	if sep == "" {
		return " · "
	}
	return sep
}

// MaxFieldsPerLineOrDefault returns how many fields fit on one line before wrapping.
func (c TurnMetricsUIConfig) MaxFieldsPerLineOrDefault() int {
	if c.MaxFieldsPerLine > 0 {
		return c.MaxFieldsPerLine
	}
	return DefaultUIConfig().TurnMetrics.MaxFieldsPerLineOrDefault()
}

// ResolvedFields returns normalized metric field ids in display order.
func (c TurnMetricsUIConfig) ResolvedFields() []string {
	if len(c.normalized) > 0 {
		out := make([]string, len(c.normalized))
		copy(out, c.normalized)
		return out
	}
	if len(c.Fields) > 0 {
		return normalizeTurnMetricFields(c.Fields)
	}
	if preset := strings.ToLower(strings.TrimSpace(c.Preset)); preset != "" {
		if fields, ok := turnMetricsPresets[preset]; ok {
			out := make([]string, len(fields))
			copy(out, fields)
			return out
		}
	}
	return DefaultUIConfig().TurnMetrics.ResolvedFields()
}

func (c *TurnMetricsUIConfig) normalize() {
	if len(c.Fields) > 0 {
		c.normalized = normalizeTurnMetricFields(c.Fields)
		return
	}
	if preset := strings.ToLower(strings.TrimSpace(c.Preset)); preset != "" {
		if fields, ok := turnMetricsPresets[preset]; ok {
			c.normalized = append([]string(nil), fields...)
			return
		}
	}
	c.normalized = nil
}

func normalizeTurnMetricFields(fields []string) []string {
	if len(fields) == 0 {
		return DefaultUIConfig().TurnMetrics.ResolvedFields()
	}
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range fields {
		field := normalizeTurnMetricField(raw)
		if field == "" {
			continue
		}
		if _, ok := seen[field]; ok {
			continue
		}
		seen[field] = struct{}{}
		out = append(out, field)
	}
	return out
}

func normalizeTurnMetricField(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", "none", "off", "disabled":
		return ""
	case "total", "duration", "time":
		return TurnMetricFieldTotal
	case "wait", "waiting", "loading", "load", "ttft", "latency", "model":
		return TurnMetricFieldWait
	case "thought", "thinking", "think":
		return TurnMetricFieldThought
	case "generation", "writing", "write", "gen":
		return TurnMetricFieldGeneration
	case "output_tokens", "tokens", "tok", "output":
		return TurnMetricFieldOutputTokens
	case "input_tokens", "input", "in":
		return TurnMetricFieldInputTokens
	case "tokens_per_sec", "tok_s", "tok/s", "tps", "speed":
		return TurnMetricFieldTokensPerSec
	case "cost", "price", "usd":
		return TurnMetricFieldCost
	case "tools", "tool_calls", "tool":
		return TurnMetricFieldTools
	case "tool_time", "tools_time", "tool_duration":
		return TurnMetricFieldToolTime
	case "steps", "step":
		return TurnMetricFieldSteps
	case "context", "ctx":
		return TurnMetricFieldContext
	case "context_pct", "context_percent", "ctx_pct":
		return TurnMetricFieldContextPct
	default:
		return ""
	}
}
