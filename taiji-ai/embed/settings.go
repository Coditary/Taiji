package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/modelref"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

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
	UICorners               string `json:"ui_corners"`
	ToolCallDensity         string `json:"tool_call_density"`
	TurnMetricsShowActions  bool   `json:"turn_metrics_show_actions"`
	CodeBlockWrap           bool   `json:"code_block_wrap"`
	ChatWidth               string `json:"chat_width"`
	ReduceMotion            bool   `json:"reduce_motion"`
	MessageTimestamps       bool   `json:"message_timestamps"`
	MessageTimeClock        string `json:"message_time_clock"`
	StickyUserMessage       bool   `json:"sticky_user_message"`

	MCPAvailable []string `json:"mcp_available"`
	MCPEnabled   []string `json:"mcp_enabled"`
	SkillsAvail  []string `json:"skills_available"`
	SkillsAllow  []string `json:"skills_allow"`
	SkillsDeny   []string `json:"skills_deny"`

	DecisionsEnabled bool `json:"decisions_enabled"`
	MemoryEnabled    bool `json:"memory_enabled"`

	// Project scope (read/write when scope=project)
	ProjectDefaultProvider string `json:"project_default_provider,omitempty"`
	ProjectMaxSteps        int    `json:"project_max_steps,omitempty"`
	ProjectChainOfThought  string `json:"project_chain_of_thought,omitempty"`

	Templates         []session.TemplateSummary `json:"templates"`
	ApplyTemplateID   string                    `json:"apply_template_id,omitempty"`

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
	}
	if svc != nil && svc.Skills != nil {
		if all, err := svc.Skills.LoadAll(); err == nil {
			for _, sk := range all {
				out.SkillsAvail = append(out.SkillsAvail, sk.Name)
			}
			sort.Strings(out.SkillsAvail)
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
		out.MCPEnabled = append([]string(nil), st.MCPServers...)
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
	if out.ToolLabelsLocale == "" {
		out.ToolLabelsLocale = strings.TrimSpace(h.cfg.UI.ToolLabels.Locale)
		if out.ToolLabelsLocale == "" {
			out.ToolLabelsLocale = "en"
		}
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
	corners := strings.TrimSpace(ui.CornerStyle)
	if corners == "" {
		corners = "square"
	}
	out.UICorners = corners
	out.TurnMetricsShowActions = ui.TurnMetrics.ShowActionsOrDefault()
	if ui.TurnMetrics.CompactOrDefault() {
		out.ToolCallDensity = "compact"
	} else {
		out.ToolCallDensity = "detailed"
	}
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
	compact := strings.EqualFold(strings.TrimSpace(payload.ToolCallDensity), "compact")
	cfg.UI.TurnMetrics.Compact = &compact
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
	st.ToolLabelsLocale = strings.TrimSpace(payload.ToolLabelsLocale)
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
	st.MCPServers = normalizeList(payload.MCPEnabled)

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
