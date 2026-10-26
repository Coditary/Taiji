package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/modelref"
	"github.com/coditary/taiji/internal/packs"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

type mcpToolPayload struct {
	Server         string `json:"server"`
	Name           string `json:"name"`
	RegisteredName string `json:"registered_name"`
	Namespace      string `json:"namespace"`
	Description    string `json:"description"`
}

type settingsPayload struct {
	Scope string `json:"scope"`

	SessionName       string   `json:"session_name"`
	TemplateID        string   `json:"template_id"`
	Preset            string   `json:"preset"`
	Provider          string   `json:"provider"`
	ProviderOptions   []string `json:"provider_options"`
	Model             string   `json:"model"`
	ModelDefault      string   `json:"model_default"`
	Temperature         float64 `json:"temperature"`
	TemperatureSet      bool    `json:"temperature_set"`
	TemperatureDefault  float64 `json:"temperature_default"`
	MaxTokens           int     `json:"max_tokens"`
	MaxTokensSet        bool    `json:"max_tokens_set"`
	MaxTokensDefault    int     `json:"max_tokens_default"`
	ContextWindow       int      `json:"context_window"`
	ContextWindowSet    bool     `json:"context_window_set"`
	ContextWindowDefault int     `json:"context_window_default"`
	ProviderLabels      []string `json:"provider_labels"`
	ProviderModels      []string `json:"provider_models"`
	ProviderIsLocal     bool     `json:"provider_is_local"`
	MaxSteps          int      `json:"max_steps"`
	ChainOfThought    string   `json:"chain_of_thought"`
	ThoughtMaxChars   int      `json:"thought_max_chars"`
	SystemPromptExtra string   `json:"system_prompt_extra"`
	Presentation      string   `json:"presentation"`
	TurnMetricsPreset string `json:"turn_metrics_preset"`
	ToolLabelsLocale  string `json:"tool_labels_locale"`

	UITheme                 string `json:"ui_theme"`
	UILocale                string `json:"ui_locale"`
	UICorners               string `json:"ui_corners"`
	ToolCallDensity         string `json:"tool_call_density"`
	FileDiffLayout          string `json:"file_diff_layout"`
	ActivitySpacingLines    int    `json:"activity_spacing_lines"`
	MessageSpacingLines        int  `json:"message_spacing_lines"`
	ActivityShowPipelineTitle  bool   `json:"activity_show_pipeline_title"`
	ToolExpandShowParameters bool   `json:"tool_expand_show_parameters"`
	ToolExpandShowResult     bool   `json:"tool_expand_show_result"`
	ToolExpandArgsFormat     string `json:"tool_expand_args_format"`
	ToolExpandResultFormat   string `json:"tool_expand_result_format"`
	TurnMetricsShowActions  bool   `json:"turn_metrics_show_actions"`
	CodeBlockWrap           bool   `json:"code_block_wrap"`
	ChatWidth               string `json:"chat_width"`
	ReduceMotion            bool   `json:"reduce_motion"`
	MessageTimestamps       bool   `json:"message_timestamps"`
	MessageTimeClock        string `json:"message_time_clock"`
	StickyUserMessage       bool   `json:"sticky_user_message"`
	MessageFillBackground            bool `json:"message_fill_background"`
	UserMessageFillBackground        bool `json:"user_message_fill_background"`
	AssistantMessageFillBackground   bool `json:"assistant_message_fill_background"`
	UserMessageStyle          string `json:"user_message_style"`
	UserMessageAlign          string `json:"user_message_align"`
	UserMessageMaxWidthPct    int    `json:"user_message_max_width_pct"`
	AssistantMessageStyle     string `json:"assistant_message_style"`
	AssistantMessageAlign     string `json:"assistant_message_align"`
	AssistantMessageMaxWidthPct int  `json:"assistant_message_max_width_pct"`
	AutoCompactContextPct   int    `json:"auto_compact_context_pct"`
	KeyOpenThemePicker      string `json:"key_open_theme_picker"`
	KeybindingStyle         string `json:"keybinding_style"`
	VimEscapeSequence       string `json:"vim_escape_sequence"`
	LeaderKey               string `json:"leader_key"`

	MCPAvailable []string         `json:"mcp_available"`
	MCPEnabled   []string         `json:"mcp_enabled"`
	MCPTools     []mcpToolPayload `json:"mcp_tools"`
	MCPToolsDeny []string         `json:"mcp_tools_deny"`
	SkillsAvail  []string `json:"skills_available"`
	SkillsAllow  []string `json:"skills_allow"`
	SkillsDeny   []string `json:"skills_deny"`
	PacksAvail       []string `json:"packs_available"`
	PacksNodeCounts  []int    `json:"packs_node_counts"`
	PacksAllow       []string `json:"packs_allow"`

	DecisionsEnabled bool `json:"decisions_enabled"`
	MemoryEnabled    bool `json:"memory_enabled"`

	// Project scope (read/write when scope=project)
	ProjectDefaultProvider string `json:"project_default_provider,omitempty"`
	ProjectMaxSteps        int    `json:"project_max_steps,omitempty"`
	ProjectChainOfThought  string `json:"project_chain_of_thought,omitempty"`

	Templates         []session.TemplateSummary `json:"templates"`
	ApplyTemplateID   string                    `json:"apply_template_id,omitempty"`

	Avatar              string            `json:"avatar,omitempty"`
	ThemeColor          string            `json:"theme_color,omitempty"`
	TemplateDisplayName string            `json:"template_display_name,omitempty"`
	ActorState          map[string]string `json:"actor_state,omitempty"`
	ActorStateSummary   string            `json:"actor_state_summary,omitempty"`
	StreamPacing        string            `json:"stream_pacing,omitempty"`
	CouncilEnabled      bool              `json:"council_enabled,omitempty"`
	CouncilActors       []string          `json:"council_actors,omitempty"`

	BuiltinToolsAvailable []string `json:"builtin_tools_available,omitempty"`
	BuiltinToolsAllow     []string `json:"builtin_tools_allow,omitempty"`
	BuiltinToolsDeny      []string `json:"builtin_tools_deny,omitempty"`

	// models.dev capability hints for the selected provider model (settings UI visibility).
	ModelCatalogKnown       bool `json:"model_catalog_known"`
	ModelCatalogTemperature bool `json:"model_catalog_temperature"`
	ModelCatalogContext     int  `json:"model_catalog_context"`
	ModelCatalogOutput      int  `json:"model_catalog_output"`

	STTEnabled        bool     `json:"stt_enabled"`
	STTDriver         string   `json:"stt_driver"`
	STTModel          string   `json:"stt_model"`
	STTLanguage       string   `json:"stt_language"`
	STTQuality        string   `json:"stt_quality"`
	STTTrimSilence    bool     `json:"stt_trim_silence"`
	STTSilenceLevel   float32  `json:"stt_silence_level"`
	STTCaptureDevice  string   `json:"stt_capture_device"`
	STTSampleRate     int      `json:"stt_sample_rate"`
	STTDriverOptions  []string `json:"stt_driver_options"`
	STTDriverLabels   []string `json:"stt_driver_labels"`
	STTDriversError   string   `json:"stt_drivers_error,omitempty"`
	STTRecording      bool     `json:"stt_recording"`
}

func buildSettingsPayload(h *runtimeHandle, scope string) settingsPayload {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		scope = "chat"
	}
	out := settingsPayload{Scope: scope}
	if h == nil || h.cfg == nil {
		return out
	}

	providerIDs := make([]string, 0, len(h.cfg.Providers))
	for id := range h.cfg.Providers {
		providerIDs = append(providerIDs, id)
	}
	sort.Strings(providerIDs)
	out.ProviderOptions = providerIDs
	for _, id := range providerIDs {
		out.ProviderLabels = append(out.ProviderLabels, providerEngineLabel(h.cfg, id))
		if pc, ok := h.cfg.Providers[id]; ok {
			out.ProviderModels = append(out.ProviderModels, modelref.Display(strings.TrimSpace(pc.Model)))
		} else {
			out.ProviderModels = append(out.ProviderModels, "")
		}
	}

	svc, _ := h.getRuntime()
	if svc != nil && svc.MCP != nil {
		out.MCPAvailable = svc.MCP.ConnectedServers()
		sort.Strings(out.MCPAvailable)
		if catalog, err := svc.MCP.ToolCatalog(context.Background()); err == nil {
			for _, entry := range catalog {
				out.MCPTools = append(out.MCPTools, mcpToolPayload{
					Server:         entry.Server,
					Name:           entry.Name,
					RegisteredName: tools.Namespaced(entry.Server, entry.Name),
					Namespace:      entry.Server,
					Description:    entry.Description,
				})
			}
		}
	}
	if svc != nil && svc.Skills != nil {
		if all, err := svc.Skills.LoadAll(); err == nil {
			for _, sk := range all {
				out.SkillsAvail = append(out.SkillsAvail, sk.Name)
			}
			sort.Strings(out.SkillsAvail)
		}
	}
	if svc != nil && svc.PackIndex != nil && svc.PackIndex.Enabled() {
		summaries := packs.Summarize(svc.PackIndex.Nodes())
		sort.Slice(summaries, func(i, j int) bool { return summaries[i].Name < summaries[j].Name })
		for _, summary := range summaries {
			out.PacksAvail = append(out.PacksAvail, summary.Name)
			out.PacksNodeCounts = append(out.PacksNodeCounts, summary.NodeCount)
		}
	}

	rt := session.AgentRuntime{}
	if h.session != nil {
		rt = h.session.AgentRuntimeForRun(h.cfg)
		out.SessionName = strings.TrimSpace(h.session.State.Name)
		st := h.session.State.Settings
		out.TemplateID = strings.TrimSpace(st.TemplateID)
		out.Preset = strings.TrimSpace(st.Preset)
		out.Provider = h.session.ResolveProviderID(h.cfg)
		out.Model = modelref.Display(strings.TrimSpace(st.Model))
		out.SystemPromptExtra = strings.TrimSpace(st.SystemPromptExtra)
		out.Presentation = h.session.PresentationMode()
		out.TurnMetricsPreset = strings.TrimSpace(st.TurnMetricsPreset)
		out.ToolLabelsLocale = strings.TrimSpace(st.ToolLabelsLocale)
		out.SkillsAllow = append([]string(nil), st.SkillsAllow...)
		out.SkillsDeny = append([]string(nil), st.SkillsDeny...)
		out.PacksAllow = append([]string(nil), st.PacksAllow...)
		out.MCPEnabled = append([]string(nil), st.MCPServers...)
		out.MCPToolsDeny = append([]string(nil), st.MCPToolsDeny...)
		out.Avatar = strings.TrimSpace(st.Avatar)
		out.ThemeColor = strings.TrimSpace(st.ThemeColor)
		out.TemplateDisplayName = strings.TrimSpace(st.TemplateDisplayName)
		if len(st.ActorState) > 0 {
			out.ActorState = map[string]string{}
			for k, v := range st.ActorState {
				out.ActorState[k] = v
			}
			out.ActorStateSummary = session.FormatActorState(st.ActorState)
		}
		out.StreamPacing = strings.TrimSpace(st.StreamPacing)
		out.CouncilEnabled = st.CouncilEnabled
		out.CouncilActors = append([]string(nil), st.CouncilActors...)
		out.BuiltinToolsAllow = append([]string(nil), st.BuiltinToolsAllow...)
		out.BuiltinToolsDeny = append([]string(nil), st.BuiltinToolsDeny...)
		if st.Temperature != nil {
			out.Temperature = *st.Temperature
			out.TemperatureSet = true
		}
		if st.MaxTokens != nil {
			out.MaxTokens = *st.MaxTokens
			out.MaxTokensSet = true
		}
		if st.DecisionsEnabled != nil {
			out.DecisionsEnabled = *st.DecisionsEnabled
		} else {
			out.DecisionsEnabled = rt.DecisionsEnabled
		}
		if st.MemoryEnabled != nil {
			out.MemoryEnabled = *st.MemoryEnabled
		} else {
			out.MemoryEnabled = rt.MemoryEnabled
		}
	} else {
		out.DecisionsEnabled = rt.DecisionsEnabled
		out.MemoryEnabled = rt.MemoryEnabled
		out.Provider = strings.TrimSpace(h.cfg.DefaultProvider)
		out.Presentation = "markdown"
	}

	out.MaxSteps = rt.MaxSteps
	out.ChainOfThought = string(rt.ChainOfThought.OrDefault())
	out.ThoughtMaxChars = rt.ThoughtMaxChars

	providerID := out.Provider
	out.ProviderIsLocal = providerIsLocalEngine(h.cfg, providerID)
	if pc, ok := h.cfg.Providers[providerID]; ok {
		out.ModelDefault = modelref.Display(strings.TrimSpace(pc.Model))
		if out.Model == "" {
			out.Model = out.ModelDefault
		}
		if !out.TemperatureSet && pc.Temperature > 0 {
			out.Temperature = pc.Temperature
		}
		if !out.MaxTokensSet && pc.MaxTokens > 0 {
			out.MaxTokens = pc.MaxTokens
		}
		if pc.Temperature > 0 {
			out.TemperatureDefault = pc.Temperature
		}
		if pc.MaxTokens > 0 {
			out.MaxTokensDefault = pc.MaxTokens
		}
		out.ContextWindowDefault = session.ResolveContextWindow(h.cfg, providerID)
		if h.session != nil {
			if v := h.session.State.Settings.ContextWindow; v != nil && *v > 0 {
				out.ContextWindow = *v
				out.ContextWindowSet = true
			}
		}
		if !out.ContextWindowSet {
			out.ContextWindow = out.ContextWindowDefault
		}
	}
	if out.ContextWindowDefault <= 0 {
		out.ContextWindowDefault = session.ResolveContextWindow(h.cfg, providerID)
	}
	if out.ContextWindow <= 0 {
		out.ContextWindow = out.ContextWindowDefault
	}
	if out.TemperatureDefault <= 0 {
		out.TemperatureDefault = 0.7
	}
	if out.MaxTokensDefault <= 0 {
		out.MaxTokensDefault = 4096
	}
	if out.Presentation == "" {
		out.Presentation = "markdown"
	}
	if out.TurnMetricsPreset == "" {
		out.TurnMetricsPreset = strings.TrimSpace(h.cfg.UI.TurnMetrics.Preset)
		if out.TurnMetricsPreset == "" {
			out.TurnMetricsPreset = "dev"
		}
	}
	out.ToolLabelsLocale = out.UILocale
	if out.ToolLabelsLocale == "" {
		out.ToolLabelsLocale = "en"
	}
	if len(out.MCPEnabled) == 0 {
		out.MCPEnabled = append([]string(nil), out.MCPAvailable...)
	}

	out.ProjectDefaultProvider = strings.TrimSpace(h.cfg.DefaultProvider)
	out.ProjectMaxSteps = h.cfg.Agent.MaxSteps
	out.ProjectChainOfThought = string(h.cfg.Agent.ChainOfThought.OrDefault())
	out.Templates = session.ListChatTemplates(h.cfg.Root)
	fillModelCatalogHints(h, providerID, out.Model, out.ModelDefault, &out)
	fillAppearancePrefs(h, &out)
	fillSTTSettings(h, &out)
	out.STTRecording = sttRecording(h)
	if svc != nil && svc.Tools != nil {
		out.BuiltinToolsAvailable = listBuiltinToolNames(svc.Tools, out.MCPTools)
	}
	return out
}

func fillAppearancePrefs(h *runtimeHandle, out *settingsPayload) {
	if h == nil || h.cfg == nil || out == nil {
		return
	}
	ui := effectiveUI(h)
	theme := strings.TrimSpace(ui.Theme)
	if theme == "" {
		theme = "dark"
	}
	out.UITheme = theme
	out.UILocale = ui.LocaleOrDefault()
	corners := strings.TrimSpace(ui.CornerStyle)
	if corners == "" {
		corners = "square"
	}
	out.UICorners = corners
	out.TurnMetricsShowActions = ui.TurnMetrics.ShowActionsOrDefault()
	out.ToolCallDensity = ui.ToolCallDensityOrDefault()
	out.FileDiffLayout = ui.FileDiffLayoutOrDefault()
	out.ActivitySpacingLines = ui.ActivitySpacingLinesOrDefault()
	out.MessageSpacingLines = ui.MessageSpacingLinesOrDefault()
	out.ActivityShowPipelineTitle = ui.ActivityShowPipelineTitleOrDefault()
	out.ToolExpandShowParameters = ui.ToolExpandShowParametersOrDefault()
	out.ToolExpandShowResult = ui.ToolExpandShowResultOrDefault()
	out.ToolExpandArgsFormat = ui.ToolExpandArgsFormatOrDefault()
	out.ToolExpandResultFormat = ui.ToolExpandResultFormatOrDefault()
	if ui.CodeBlockWrap != nil {
		out.CodeBlockWrap = *ui.CodeBlockWrap
	} else {
		out.CodeBlockWrap = true
	}
	switch ui.ChatMaxWidth {
	case 0:
		out.ChatWidth = "auto"
	case 88:
		out.ChatWidth = "narrow"
	case 120:
		out.ChatWidth = "medium"
	default:
		if ui.ChatMaxWidth > 0 {
			out.ChatWidth = "wide"
		} else {
			out.ChatWidth = "auto"
		}
	}
	if ui.ReduceMotion != nil {
		out.ReduceMotion = *ui.ReduceMotion
	}
	out.MessageTimestamps = ui.MessageTimestampsOrDefault()
	out.MessageTimeClock = ui.MessageTimeClockOrDefault()
	out.StickyUserMessage = ui.StickyUserMessageOrDefault()
	out.MessageFillBackground = ui.MessageFillBackgroundOrDefault()
	out.UserMessageFillBackground = ui.UserMessageFillBackgroundOrDefault()
	out.AssistantMessageFillBackground = ui.AssistantMessageFillBackgroundOrDefault()
	out.UserMessageStyle = ui.UserMessageStyleOrDefault()
	out.UserMessageAlign = ui.UserMessageAlignOrDefault()
	out.UserMessageMaxWidthPct = ui.UserMessageMaxWidthPctOrDefault()
	out.AssistantMessageStyle = ui.AssistantMessageStyleOrDefault()
	out.AssistantMessageAlign = ui.AssistantMessageAlignOrDefault()
	out.AssistantMessageMaxWidthPct = ui.AssistantMessageMaxWidthPctOrDefault()
	out.AutoCompactContextPct = ui.AutoCompactContextPct
	kb := ui.KeybindingsOrDefault()
	out.KeyOpenThemePicker = kb.OpenThemePicker
	out.KeybindingStyle = ui.KeybindingStyleOrDefault()
	out.VimEscapeSequence = ui.VimEscapeSequenceOrDefault()
	out.LeaderKey = ui.LeaderKeyOrDefault()
}

func applyAppearancePrefs(cfg *taijicfg.Config, payload settingsPayload) {
	if cfg == nil {
		return
	}
	theme := strings.TrimSpace(payload.UITheme)
	if theme == "" {
		theme = "dark"
	}
	cfg.UI.Theme = theme
	cfg.UI.Locale = taijicfg.NormalizeUILocale(payload.UILocale)
	cfg.UI.ToolLabels.Locale = taijicfg.NormalizeUILocale(payload.UILocale)
	corners := strings.TrimSpace(payload.UICorners)
	if corners == "" {
		corners = "square"
	}
	switch strings.ToLower(corners) {
	case "rounded":
		corners = "rounded"
	case "flat", "filled", "minimal":
		corners = "flat"
	default:
		corners = "square"
	}
	cfg.UI.CornerStyle = corners
	show := payload.TurnMetricsShowActions
	cfg.UI.TurnMetrics.ShowActions = &show
	cfg.UI.ToolCallDensity = taijicfg.NormalizeToolCallDensity(payload.ToolCallDensity)
	cfg.UI.FileDiffLayout = taijicfg.NormalizeFileDiffLayout(payload.FileDiffLayout)
	cfg.UI.ActivitySpacingLines = taijicfg.ClampSpacingLines(payload.ActivitySpacingLines)
	cfg.UI.MessageSpacingLines = taijicfg.ClampSpacingLines(payload.MessageSpacingLines)
	showPipeline := payload.ActivityShowPipelineTitle
	cfg.UI.ActivityShowPipelineTitle = &showPipeline
	showToolParams := payload.ToolExpandShowParameters
	cfg.UI.ToolExpandShowParameters = &showToolParams
	showToolResult := payload.ToolExpandShowResult
	cfg.UI.ToolExpandShowResult = &showToolResult
	cfg.UI.ToolExpandArgsFormat = taijicfg.NormalizeToolExpandArgsFormat(payload.ToolExpandArgsFormat)
	cfg.UI.ToolExpandResultFormat = taijicfg.NormalizeToolExpandResultFormat(payload.ToolExpandResultFormat)
	wrap := payload.CodeBlockWrap
	cfg.UI.CodeBlockWrap = &wrap
	switch strings.TrimSpace(payload.ChatWidth) {
	case "narrow":
		cfg.UI.ChatMaxWidth = 88
	case "medium":
		cfg.UI.ChatMaxWidth = 120
	case "wide":
		cfg.UI.ChatMaxWidth = 160
	default:
		cfg.UI.ChatMaxWidth = 0
	}
	motion := payload.ReduceMotion
	cfg.UI.ReduceMotion = &motion
	ts := payload.MessageTimestamps
	cfg.UI.MessageTimestamps = &ts
	cfg.UI.MessageTimeClock = taijicfg.UIConfig{MessageTimeClock: payload.MessageTimeClock}.MessageTimeClockOrDefault()
	sticky := payload.StickyUserMessage
	cfg.UI.StickyUserMessage = &sticky
	msgFill := payload.MessageFillBackground
	cfg.UI.MessageFillBackground = &msgFill
	userFill := payload.UserMessageFillBackground
	cfg.UI.UserMessageFillBackground = &userFill
	asstFill := payload.AssistantMessageFillBackground
	cfg.UI.AssistantMessageFillBackground = &asstFill
	cfg.UI.UserMessageStyle = taijicfg.NormalizeUserMessageStyle(payload.UserMessageStyle)
	cfg.UI.UserMessageAlign = taijicfg.NormalizeMessageAlign(payload.UserMessageAlign)
	cfg.UI.UserMessageMaxWidthPct = taijicfg.UIConfig{UserMessageMaxWidthPct: payload.UserMessageMaxWidthPct}.UserMessageMaxWidthPctOrDefault()
	cfg.UI.AssistantMessageStyle = taijicfg.NormalizeUserMessageStyle(payload.AssistantMessageStyle)
	cfg.UI.AssistantMessageAlign = taijicfg.NormalizeMessageAlign(payload.AssistantMessageAlign)
	cfg.UI.AssistantMessageMaxWidthPct =
		taijicfg.UIConfig{AssistantMessageMaxWidthPct: payload.AssistantMessageMaxWidthPct}.AssistantMessageMaxWidthPctOrDefault()
	if payload.AutoCompactContextPct < 0 {
		cfg.UI.AutoCompactContextPct = 0
	} else if payload.AutoCompactContextPct > 100 {
		cfg.UI.AutoCompactContextPct = 100
	} else {
		cfg.UI.AutoCompactContextPct = payload.AutoCompactContextPct
	}
	kb := taijicfg.NormalizeUIShortcut(payload.KeyOpenThemePicker)
	if kb == "" {
		kb = "Ctrl+T"
	}
	cfg.UI.Keybindings.OpenThemePicker = kb
	cfg.UI.KeybindingStyle = taijicfg.NormalizeKeybindingStyle(payload.KeybindingStyle)
	cfg.UI.VimEscapeSequence = taijicfg.NormalizeVimEscapeSequence(payload.VimEscapeSequence)
	cfg.UI.LeaderKey = taijicfg.NormalizeLeaderKey(payload.LeaderKey)
}

func fillModelCatalogHints(h *runtimeHandle, providerID, model, modelDefault string, out *settingsPayload) {
	if h == nil || h.cfg == nil || out == nil {
		return
	}
	providerID = strings.TrimSpace(providerID)
	candidates := modelref.Candidates(h.cfg, providerID)
	modelCanon := modelref.Expand(strings.TrimSpace(model), candidates)
	if modelCanon == "" {
		modelCanon = modelref.Expand(strings.TrimSpace(modelDefault), candidates)
	}
	if modelCanon == "" {
		modelCanon = strings.TrimSpace(model)
	}
	if modelCanon == "" {
		return
	}
	details, err := provider.GetProviderModelDetails(context.Background(), h.cfg, providerID, modelCanon)
	if err != nil {
		return
	}
	out.ModelCatalogKnown = details.CatalogKnown
	out.ModelCatalogTemperature = details.Temperature
	out.ModelCatalogContext = details.Context
	out.ModelCatalogOutput = details.Output
}

func stripUnsupportedModelSettings(h *runtimeHandle, providerID, model, modelDefault string, st *session.Settings) {
	if h == nil || h.cfg == nil || st == nil {
		return
	}
	var hints settingsPayload
	fillModelCatalogHints(h, providerID, model, modelDefault, &hints)
	if !hints.ModelCatalogKnown {
		return
	}
	if !hints.ModelCatalogTemperature {
		st.Temperature = nil
		st.MaxTokens = nil
	}
	if hints.ModelCatalogContext > 0 {
		v := hints.ModelCatalogContext
		st.ContextWindow = &v
	} else {
		st.ContextWindow = nil
	}
}

func applySettingsPayload(h *runtimeHandle, payload settingsPayload) error {
	if h == nil || h.cfg == nil {
		return fmt.Errorf("runtime not ready")
	}
	scope := strings.TrimSpace(payload.Scope)
	if scope == "" {
		scope = "chat"
	}
	if scope == "project" {
		if id := strings.TrimSpace(payload.ProjectDefaultProvider); id != "" {
			h.cfg.DefaultProvider = id
		}
		if payload.ProjectMaxSteps > 0 {
			h.cfg.Agent.MaxSteps = payload.ProjectMaxSteps
		}
		if mode := strings.TrimSpace(payload.ProjectChainOfThought); mode != "" {
			h.cfg.Agent.ChainOfThought = taijicfg.ChainOfThoughtMode(mode)
		}
		applyAppearancePrefs(h.cfg, payload)
		applySTTSettings(h.cfg, payload)
		return h.cfg.Save()
	}

	if h.session == nil {
		return fmt.Errorf("no active session")
	}
	if tid := strings.TrimSpace(payload.ApplyTemplateID); tid != "" {
		if err := session.ApplyChatTemplate(h.session, h.cfg.Root, tid); err != nil {
			return err
		}
	}
	st := h.session.State.Settings
	if name := strings.TrimSpace(payload.SessionName); name != "" {
		h.session.State.Name = name
		h.session.State.NameManual = true
	}
	st.TemplateID = strings.TrimSpace(payload.TemplateID)
	st.Preset = strings.TrimSpace(payload.Preset)
	st.Presentation = strings.TrimSpace(payload.Presentation)
	st.TurnMetricsPreset = strings.TrimSpace(payload.TurnMetricsPreset)
	st.ToolLabelsLocale = taijicfg.NormalizeUILocale(payload.UILocale)
	st.SystemPromptExtra = payload.SystemPromptExtra

	if id := strings.TrimSpace(payload.Provider); id != "" {
		st.Provider = id
		h.session.State.Provider = id
	}
	providerID := strings.TrimSpace(payload.Provider)
	if providerID == "" && h.session != nil {
		providerID = h.session.ResolveProviderID(h.cfg)
	}
	candidates := modelref.Candidates(h.cfg, providerID)
	defaultCanon := modelref.Expand(strings.TrimSpace(payload.ModelDefault), candidates)
	userCanon := modelref.Expand(strings.TrimSpace(payload.Model), candidates)
	if userCanon == "" || userCanon == defaultCanon ||
		modelref.Display(userCanon) == modelref.Display(defaultCanon) {
		st.Model = ""
	} else {
		st.Model = modelref.Display(userCanon)
	}

	if payload.TemperatureSet {
		v := payload.Temperature
		st.Temperature = &v
	} else {
		st.Temperature = nil
	}
	if payload.MaxTokensSet {
		v := payload.MaxTokens
		st.MaxTokens = &v
	} else {
		st.MaxTokens = nil
	}
	if payload.MaxSteps > 0 {
		v := payload.MaxSteps
		st.MaxSteps = &v
	}
	if mode := strings.TrimSpace(payload.ChainOfThought); mode != "" {
		st.ChainOfThought = mode
	}
	if payload.ThoughtMaxChars > 0 {
		v := payload.ThoughtMaxChars
		st.ThoughtMaxChars = &v
	}
	if payload.ContextWindowSet {
		v := payload.ContextWindow
		st.ContextWindow = &v
	} else {
		st.ContextWindow = nil
	}

	st.SkillsAllow = normalizeList(payload.SkillsAllow)
	st.SkillsDeny = normalizeList(payload.SkillsDeny)
	st.PacksAllow = normalizeList(payload.PacksAllow)
	st.MCPServers = normalizeList(payload.MCPEnabled)
	st.MCPToolsDeny = normalizeMCPToolsDeny(payload.MCPToolsDeny)
	st.BuiltinToolsAllow = normalizeList(payload.BuiltinToolsAllow)
	st.BuiltinToolsDeny = normalizeList(payload.BuiltinToolsDeny)
	st.StreamPacing = strings.TrimSpace(payload.StreamPacing)
	st.CouncilEnabled = payload.CouncilEnabled
	st.CouncilActors = normalizeList(payload.CouncilActors)

	dec := payload.DecisionsEnabled
	st.DecisionsEnabled = &dec
	mem := payload.MemoryEnabled
	st.MemoryEnabled = &mem

	modelDefault := ""
	if pc, ok := h.cfg.Providers[providerID]; ok {
		modelDefault = modelref.Display(strings.TrimSpace(pc.Model))
	}
	stripUnsupportedModelSettings(h, providerID, payload.Model, modelDefault, &st)

	applyAppearancePrefs(h.cfg, payload)
	applySTTSettings(h.cfg, payload)
	if err := h.cfg.Save(); err != nil {
		return err
	}

	h.session.State.Settings = st
	return h.session.Save()
}

func normalizeMCPToolsDeny(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range items {
		server, tool, ok := session.ParseMCPToolRef(raw)
		if !ok {
			continue
		}
		ref := session.MCPToolRef(server, tool)
		key := strings.ToLower(ref)
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, ref)
	}
	sort.Strings(out)
	return out
}

func normalizeList(items []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range items {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		key := strings.ToLower(v)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, v)
	}
	sort.Strings(out)
	return out
}

func providerEngineLabel(cfg *taijicfg.Config, providerKey string) string {
	if cfg == nil {
		return providerKey
	}
	pc, ok := cfg.Providers[providerKey]
	if !ok {
		return humanizeProviderID(providerKey)
	}
	if name := strings.TrimSpace(pc.DisplayName); name != "" {
		if providerIsLocalEngine(cfg, providerKey) {
			return name + " (Local)"
		}
		return name
	}
	driver := strings.TrimSpace(pc.ProviderID)
	if driver == "" {
		driver = strings.TrimSpace(pc.Type)
	}
	if driver == "" {
		driver = providerKey
	}
	label := humanizeProviderID(driver)
	if providerIsLocalEngine(cfg, providerKey) {
		return label + " (Local)"
	}
	return label
}

func providerIsLocalEngine(cfg *taijicfg.Config, providerKey string) bool {
	if cfg == nil {
		return false
	}
	pc, ok := cfg.Providers[providerKey]
	if !ok {
		return strings.Contains(strings.ToLower(providerKey), "local") ||
			strings.Contains(strings.ToLower(providerKey), "ollama")
	}
	switch strings.ToLower(strings.TrimSpace(pc.WujiKind)) {
	case "driver":
		return true
	case "api", "catalog", "provider":
		return false
	}
	driver := strings.ToLower(strings.TrimSpace(pc.ProviderID))
	switch driver {
	case "llama", "vllm", "ollama", "mlx", "koboldcpp", "localai":
		return true
	}
	t := strings.ToLower(strings.TrimSpace(pc.Type))
	switch t {
	case "wuji", "ollama", "llama", "local":
		return true
	default:
		return strings.Contains(t, "ollama") || strings.Contains(t, "llama")
	}
}

func humanizeProviderID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "Unknown"
	}
	if strings.EqualFold(id, "openrouter") {
		return "OpenRouter"
	}
	if strings.EqualFold(id, "anthropic") {
		return "Anthropic"
	}
	if strings.EqualFold(id, "ollama") {
		return "Ollama"
	}
	runes := []rune(id)
	if len(runes) == 0 {
		return id
	}
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func effectiveUI(h *runtimeHandle) taijicfg.UIConfig {
	if h == nil || h.cfg == nil {
		return taijicfg.DefaultUIConfig()
	}
	if h.session != nil {
		return h.session.UISettingsForDisplay(h.cfg)
	}
	return h.cfg.UI
}
