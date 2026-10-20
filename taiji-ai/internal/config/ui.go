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
	// MessageFillBackground fills framed chat bubbles (user + assistant) with a solid panel color.
	// When false, only the border/outline is drawn on the main chat background.
	MessageFillBackground *bool `yaml:"message_fill_background,omitempty"`
	// UserMessageStyle is how outbound user prompts render in the TUI: block, plain, messenger, header,
	// terminal, log, card, section, or cornered.
	UserMessageStyle string `yaml:"user_message_style,omitempty"`
	// UserMessageAlign is auto, left, right, or center for outbound prompts in the TUI.
	UserMessageAlign string `yaml:"user_message_align,omitempty"`
	// UserMessageMaxWidthPct caps framed message width as a percent of the chat column (40–100).
	UserMessageMaxWidthPct int `yaml:"user_message_max_width_pct,omitempty"`
	// AssistantMessageStyle is how assistant replies render in the TUI (same ids as user_message_style).
	AssistantMessageStyle string `yaml:"assistant_message_style,omitempty"`
	AssistantMessageAlign string `yaml:"assistant_message_align,omitempty"`
	AssistantMessageMaxWidthPct int `yaml:"assistant_message_max_width_pct,omitempty"`
	// AutoCompactContextPct triggers compaction before a chat run when context usage is at or above this
	// percent (0 = disabled).
	AutoCompactContextPct int `yaml:"auto_compact_context_pct,omitempty"`
	// Keybindings configures global TUI shortcuts (e.g. Ctrl+T).
	Keybindings UIKeyBindings `yaml:"keybindings,omitempty"`
}

// UIKeyBindings holds configurable keyboard shortcuts for the TUI.
type UIKeyBindings struct {
	// OpenThemePicker opens the fuzzy color-theme picker (default Ctrl+T).
	OpenThemePicker string `yaml:"open_theme_picker,omitempty"`
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
	messageFillBackground := true
	return UIConfig{
		Theme:             "dark",
		UserMessageStyle:            "messenger",
		UserMessageAlign:            "auto",
		UserMessageMaxWidthPct:      80,
		AssistantMessageStyle:       "messenger",
		AssistantMessageAlign:       "auto",
		AssistantMessageMaxWidthPct: 80,
		MessageTimestamps: &messageTimestamps,
		MessageTimeClock:  "24h",
		StickyUserMessage:           &stickyUserMessage,
		MessageFillBackground:       &messageFillBackground,
		Keybindings: UIKeyBindings{
			OpenThemePicker: "Ctrl+T",
		},
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

// MessageFillBackgroundOrDefault reports whether chat message panels use a solid fill.
func (c UIConfig) MessageFillBackgroundOrDefault() bool {
	if c.MessageFillBackground == nil {
		return true
	}
	return *c.MessageFillBackground
}

// KeybindingsOrDefault returns UI shortcuts with built-in defaults filled in.
func (c UIConfig) KeybindingsOrDefault() UIKeyBindings {
	kb := c.Keybindings
	if strings.TrimSpace(kb.OpenThemePicker) == "" {
		kb.OpenThemePicker = "Ctrl+T"
	}
	return kb
}

// NormalizeUIShortcut canonicalizes a shortcut like "ctrl+t" to "Ctrl+T".
func NormalizeUIShortcut(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	parts := strings.Split(raw, "+")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		switch strings.ToLower(part) {
		case "ctrl", "control":
			out = append(out, "Ctrl")
		case "alt", "option":
			out = append(out, "Alt")
		case "shift":
			out = append(out, "Shift")
		case "meta", "cmd", "command", "super":
			out = append(out, "Meta")
		default:
			if len(part) == 1 {
				out = append(out, strings.ToUpper(part))
			} else {
				out = append(out, part)
			}
		}
	}
	return strings.Join(out, "+")
}

// CornersRoundedOrDefault reports whether framed UI chrome uses rounded box corners.
func (c UIConfig) CornersRoundedOrDefault() bool {
	return strings.EqualFold(strings.TrimSpace(c.CornerStyle), "rounded")
}

// UserMessageStyleOrDefault returns the normalized user message chrome id.
func (c UIConfig) UserMessageStyleOrDefault() string {
	return NormalizeUserMessageStyle(c.UserMessageStyle)
}

// UserMessageAlignOrDefault returns auto, left, right, or center.
func (c UIConfig) UserMessageAlignOrDefault() string {
	return NormalizeMessageAlign(c.UserMessageAlign)
}

// UserMessageMaxWidthPctOrDefault returns 40–100 (default 80).
func (c UIConfig) UserMessageMaxWidthPctOrDefault() int {
	return clampMessageWidthPct(c.UserMessageMaxWidthPct)
}

// AssistantMessageStyleOrDefault returns the normalized assistant message chrome id.
func (c UIConfig) AssistantMessageStyleOrDefault() string {
	if strings.TrimSpace(c.AssistantMessageStyle) == "" {
		return "messenger"
	}
	return NormalizeUserMessageStyle(c.AssistantMessageStyle)
}

// AssistantMessageAlignOrDefault returns auto, left, right, or center.
func (c UIConfig) AssistantMessageAlignOrDefault() string {
	return NormalizeMessageAlign(c.AssistantMessageAlign)
}

// AssistantMessageMaxWidthPctOrDefault returns 40–100 (default 80).
func (c UIConfig) AssistantMessageMaxWidthPctOrDefault() int {
	return clampMessageWidthPct(c.AssistantMessageMaxWidthPct)
}

// NormalizeMessageAlign maps settings values to a supported alignment id.
func NormalizeMessageAlign(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "left", "l":
		return "left"
	case "right", "r":
		return "right"
	case "center", "centre", "middle", "c":
		return "center"
	default:
		return "auto"
	}
}

func clampMessageWidthPct(pct int) int {
	if pct <= 0 {
		return 80
	}
	if pct < 40 {
		return 40
	}
	if pct > 100 {
		return 100
	}
	return pct
}

// NormalizeUserMessageStyle maps settings values to a supported user message style id.
func NormalizeUserMessageStyle(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "plain", "inline", "text":
		return "plain"
	case "messenger", "bubble", "chat":
		return "messenger"
	case "header", "header_strip", "title":
		return "header"
	case "terminal", "prompt", "cli":
		return "terminal"
	case "log":
		return "log"
	case "card", "pill", "chip":
		return "card"
	case "section", "divider", "rule":
		return "section"
	case "cornered", "corners", "corner_box":
		return "cornered"
	case "block", "framed", "card_block":
		return "block"
	case "":
		return "messenger"
	default:
		return "messenger"
	}
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
