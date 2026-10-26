package config

import "strings"

// UIConfig controls Taiji TUI presentation.
type UIConfig struct {
	TurnMetrics TurnMetricsUIConfig `yaml:"turn_metrics"`
	ToolLabels  ToolLabelsUIConfig  `yaml:"tool_labels"`
	// Theme is a taiji-tui stylesheet id (built-in: dark, high_contrast; or ~/.config/taiji/themes/<id>.theme).
	Theme string `yaml:"theme,omitempty"`
	// Locale is the TUI interface language id (en, de, es, …).
	Locale string `yaml:"locale,omitempty"`
	// CornerStyle is "square", "rounded", or "flat" (fill only, no outline) for composer and user bubbles.
	CornerStyle string `yaml:"corner_style,omitempty"`
	// ToolCallDensity controls the TUI activity log: inline, subtree, clustered, or ignore.
	ToolCallDensity string `yaml:"tool_call_density,omitempty"`
	// FileDiffLayout is stacked (unified) or side_by_side for file change cards in the TUI.
	FileDiffLayout string `yaml:"file_diff_layout,omitempty"`
	// ActivitySpacingLines is blank lines between tool/diff rows in the activity log (0–4).
	ActivitySpacingLines int `yaml:"activity_spacing_lines,omitempty"`
	// MessageSpacingLines is blank lines between user prompts and replies / turns (0–4).
	MessageSpacingLines int `yaml:"message_spacing_lines,omitempty"`
	// ActivityShowPipelineTitle shows the "Taiji (model)" row above tool activity in the TUI.
	ActivityShowPipelineTitle *bool `yaml:"activity_show_pipeline_title,omitempty"`
	// ToolExpandShowParameters shows tool arguments when a tool row is expanded in the TUI.
	ToolExpandShowParameters *bool `yaml:"tool_expand_show_parameters,omitempty"`
	// ToolExpandShowResult shows tool output when a tool row is expanded in the TUI.
	ToolExpandShowResult *bool `yaml:"tool_expand_show_result,omitempty"`
	// ToolExpandArgsFormat is pretty (key: value) or json (raw) for expanded parameters.
	ToolExpandArgsFormat string `yaml:"tool_expand_args_format,omitempty"`
	// ToolExpandResultFormat is text (wrapped) or json (pretty when valid JSON) for expanded results.
	ToolExpandResultFormat string `yaml:"tool_expand_result_format,omitempty"`
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
	// MessageFillBackground is deprecated; use UserMessageFillBackground and AssistantMessageFillBackground.
	MessageFillBackground *bool `yaml:"message_fill_background,omitempty"`
	// UserMessageFillBackground fills the user's framed bubble (border included) with a solid panel color.
	UserMessageFillBackground *bool `yaml:"user_message_fill_background,omitempty"`
	// AssistantMessageFillBackground fills assistant framed bubbles (border included).
	AssistantMessageFillBackground *bool `yaml:"assistant_message_fill_background,omitempty"`
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
	// KeybindingStyle is "standard" or "vim" (modal navigation).
	KeybindingStyle string `yaml:"keybinding_style,omitempty"`
	// VimEscapeSequence leaves insert mode in vim style (default jk).
	VimEscapeSequence string `yaml:"vim_escape_sequence,omitempty"`
	// LeaderKey is the modal leader key name (default Space).
	LeaderKey string `yaml:"leader_key,omitempty"`
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
	userMessageFillBackground := true
	assistantMessageFillBackground := true
	activityShowPipelineTitle := true
	toolExpandShowParameters := true
	toolExpandShowResult := true
	return UIConfig{
		Theme:             "dark",
		Locale:            "en",
		UserMessageStyle:            "messenger",
		UserMessageAlign:            "auto",
		UserMessageMaxWidthPct:      80,
		AssistantMessageStyle:       "messenger",
		AssistantMessageAlign:       "auto",
		AssistantMessageMaxWidthPct: 80,
		MessageTimestamps: &messageTimestamps,
		MessageTimeClock:  "24h",
		StickyUserMessage:           &stickyUserMessage,
		MessageFillBackground:             &messageFillBackground,
		UserMessageFillBackground:         &userMessageFillBackground,
		AssistantMessageFillBackground:    &assistantMessageFillBackground,
		Keybindings: UIKeyBindings{
			OpenThemePicker: "Ctrl+T",
		},
		ToolCallDensity:   "subtree",
		FileDiffLayout:        "stacked",
		ActivitySpacingLines:  1,
		MessageSpacingLines:        1,
		ActivityShowPipelineTitle:  &activityShowPipelineTitle,
		ToolExpandShowParameters:   &toolExpandShowParameters,
		ToolExpandShowResult:       &toolExpandShowResult,
		ToolExpandArgsFormat:       "pretty",
		ToolExpandResultFormat:     "text",
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

// MessageFillBackgroundOrDefault reports whether chat message panels use a solid fill (legacy).
func (c UIConfig) MessageFillBackgroundOrDefault() bool {
	if c.MessageFillBackground == nil {
		return true
	}
	return *c.MessageFillBackground
}

func (c UIConfig) UserMessageFillBackgroundOrDefault() bool {
	if c.UserMessageFillBackground != nil {
		return *c.UserMessageFillBackground
	}
	return c.MessageFillBackgroundOrDefault()
}

func (c UIConfig) AssistantMessageFillBackgroundOrDefault() bool {
	if c.AssistantMessageFillBackground != nil {
		return *c.AssistantMessageFillBackground
	}
	return c.MessageFillBackgroundOrDefault()
}

// KeybindingsOrDefault returns UI shortcuts with built-in defaults filled in.
func (c UIConfig) KeybindingsOrDefault() UIKeyBindings {
	kb := c.Keybindings
	if strings.TrimSpace(kb.OpenThemePicker) == "" {
		kb.OpenThemePicker = "Ctrl+T"
	}
	return kb
}

func (c UIConfig) KeybindingStyleOrDefault() string {
	return NormalizeKeybindingStyle(c.KeybindingStyle)
}

func (c UIConfig) VimEscapeSequenceOrDefault() string {
	return NormalizeVimEscapeSequence(c.VimEscapeSequence)
}

func (c UIConfig) LeaderKeyOrDefault() string {
	return NormalizeLeaderKey(c.LeaderKey)
}

// NormalizeKeybindingStyle returns "standard" or "vim".
func NormalizeKeybindingStyle(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "vim", "modal", "vim_modal":
		return "vim"
	default:
		return "standard"
	}
}

// NormalizeVimEscapeSequence returns a non-empty escape chord (default jk).
func NormalizeVimEscapeSequence(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "jk"
	}
	return raw
}

// NormalizeLeaderKey returns a non-empty leader key label (default Space).
func NormalizeLeaderKey(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "Space"
	}
	return raw
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

// NormalizeToolCallDensity maps stored / legacy values to inline, subtree, clustered, or ignore.
func NormalizeToolCallDensity(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "inline", "subtree", "clustered", "ignore":
		return strings.ToLower(strings.TrimSpace(v))
	case "compact":
		return "clustered"
	case "detailed":
		return "subtree"
	default:
		return "subtree"
	}
}

// NormalizeFileDiffLayout maps stored values to stacked or side_by_side.
func NormalizeFileDiffLayout(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "side_by_side", "side-by-side", "split", "horizontal":
		return "side_by_side"
	default:
		return "stacked"
	}
}

// ClampSpacingLines limits chat spacing prefs to 0–4 terminal rows.
func ClampSpacingLines(n int) int {
	if n < 0 {
		return 0
	}
	if n > 4 {
		return 4
	}
	return n
}

// ActivityShowPipelineTitleOrDefault reports whether the model row is shown above tool activity.
func (c UIConfig) ActivityShowPipelineTitleOrDefault() bool {
	if c.ActivityShowPipelineTitle == nil {
		return true
	}
	return *c.ActivityShowPipelineTitle
}

// NormalizeToolExpandArgsFormat maps stored values to pretty or json.
func NormalizeToolExpandArgsFormat(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "json", "raw":
		return "json"
	default:
		return "pretty"
	}
}

// NormalizeToolExpandResultFormat maps stored values to text or json.
func NormalizeToolExpandResultFormat(v string) string {
	if strings.ToLower(strings.TrimSpace(v)) == "json" {
		return "json"
	}
	return "text"
}

// ToolExpandShowParametersOrDefault reports whether expanded tool rows show arguments.
func (c UIConfig) ToolExpandShowParametersOrDefault() bool {
	if c.ToolExpandShowParameters == nil {
		return true
	}
	return *c.ToolExpandShowParameters
}

// ToolExpandShowResultOrDefault reports whether expanded tool rows show output.
func (c UIConfig) ToolExpandShowResultOrDefault() bool {
	if c.ToolExpandShowResult == nil {
		return true
	}
	return *c.ToolExpandShowResult
}

// ToolExpandArgsFormatOrDefault returns how tool parameters render when expanded.
func (c UIConfig) ToolExpandArgsFormatOrDefault() string {
	if strings.TrimSpace(c.ToolExpandArgsFormat) != "" {
		return NormalizeToolExpandArgsFormat(c.ToolExpandArgsFormat)
	}
	return "pretty"
}

// ToolExpandResultFormatOrDefault returns how tool results render when expanded.
func (c UIConfig) ToolExpandResultFormatOrDefault() string {
	if strings.TrimSpace(c.ToolExpandResultFormat) != "" {
		return NormalizeToolExpandResultFormat(c.ToolExpandResultFormat)
	}
	return "text"
}

// ActivitySpacingLinesOrDefault returns gaps between activity-log rows.
func (c UIConfig) ActivitySpacingLinesOrDefault() int {
	if c.ActivitySpacingLines >= 0 && c.ActivitySpacingLines <= 4 {
		return c.ActivitySpacingLines
	}
	return 1
}

// MessageSpacingLinesOrDefault returns gaps between chat turns and user/assistant blocks.
func (c UIConfig) MessageSpacingLinesOrDefault() int {
	if c.MessageSpacingLines >= 0 && c.MessageSpacingLines <= 4 {
		return c.MessageSpacingLines
	}
	return 1
}

// FileDiffLayoutOrDefault returns how file diffs render in the TUI activity log.
func (c UIConfig) FileDiffLayoutOrDefault() string {
	if strings.TrimSpace(c.FileDiffLayout) != "" {
		return NormalizeFileDiffLayout(c.FileDiffLayout)
	}
	return "stacked"
}

// ToolCallDensityOrDefault returns the activity-log display mode for the TUI.
func (c UIConfig) ToolCallDensityOrDefault() string {
	if strings.TrimSpace(c.ToolCallDensity) != "" {
		return NormalizeToolCallDensity(c.ToolCallDensity)
	}
	if c.TurnMetrics.Compact != nil {
		if *c.TurnMetrics.Compact {
			return "clustered"
		}
		return "subtree"
	}
	return "subtree"
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

// NormalizeUILocale returns a supported TUI locale id or "en".
func NormalizeUILocale(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "de", "es", "fr", "it", "pt", "nl", "pl", "ru", "ja", "zh", "hi", "ar", "id", "vi", "tr":
		return strings.ToLower(strings.TrimSpace(raw))
	default:
		return "en"
	}
}

// LocaleOrDefault reports the interface language for settings labels.
func (c UIConfig) LocaleOrDefault() string {
	return NormalizeUILocale(c.Locale)
}
