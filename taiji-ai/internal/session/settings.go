package session

import (
	"sort"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// FormatActorState renders actor state for the TUI sidebar.
func FormatActorState(state map[string]string) string {
	if len(state) == 0 {
		return ""
	}
	keys := make([]string, 0, len(state))
	for k := range state {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var parts []string
	for _, k := range keys {
		parts = append(parts, k+": "+strings.TrimSpace(state[k]))
	}
	return strings.Join(parts, " · ")
}

// Settings holds per-chat overrides (only non-zero / non-empty fields apply).
type Settings struct {
	Provider          string   `json:"provider,omitempty" yaml:"provider,omitempty"`
	Model             string   `json:"model,omitempty" yaml:"model,omitempty"`
	Temperature       *float64 `json:"temperature,omitempty" yaml:"temperature,omitempty"`
	MaxTokens         *int     `json:"max_tokens,omitempty" yaml:"max_tokens,omitempty"`
	MaxSteps          *int     `json:"max_steps,omitempty" yaml:"max_steps,omitempty"`
	ChainOfThought    string   `json:"chain_of_thought,omitempty" yaml:"chain_of_thought,omitempty"`
	ThoughtMaxChars   *int     `json:"thought_max_chars,omitempty" yaml:"thought_max_chars,omitempty"`
	SystemPromptExtra string   `json:"system_prompt_extra,omitempty" yaml:"system_prompt_extra,omitempty"`
	SkillsAllow       []string `json:"skills_allow,omitempty" yaml:"skills_allow,omitempty"`
	SkillsDeny        []string `json:"skills_deny,omitempty" yaml:"skills_deny,omitempty"`
	PacksAllow        []string `json:"packs_allow,omitempty" yaml:"packs_allow,omitempty"`
	DefaultPacks      []string `json:"default_packs,omitempty" yaml:"default_packs,omitempty"`
	MCPServers        []string `json:"mcp_servers,omitempty" yaml:"mcp_servers,omitempty"`
	MCPToolsDeny      []string `json:"mcp_tools_deny,omitempty" yaml:"mcp_tools_deny,omitempty"`
	Presentation      string   `json:"presentation,omitempty" yaml:"presentation,omitempty"`
	TurnMetricsPreset string   `json:"turn_metrics_preset,omitempty" yaml:"turn_metrics_preset,omitempty"`
	ToolLabelsLocale  string   `json:"tool_labels_locale,omitempty" yaml:"tool_labels_locale,omitempty"`
	TemplateID        string   `json:"template_id,omitempty" yaml:"template_id,omitempty"`
	Preset            string   `json:"preset,omitempty" yaml:"preset,omitempty"`
	DecisionsEnabled  *bool    `json:"decisions_enabled,omitempty" yaml:"decisions_enabled,omitempty"`
	MemoryEnabled     *bool    `json:"memory_enabled,omitempty" yaml:"memory_enabled,omitempty"`
	ContextWindow     *int     `json:"context_window,omitempty" yaml:"context_window,omitempty"`

	Avatar              string            `json:"avatar,omitempty" yaml:"avatar,omitempty"`
	ThemeColor          string            `json:"theme_color,omitempty" yaml:"theme_color,omitempty"`
	TemplateDisplayName string            `json:"template_display_name,omitempty" yaml:"template_display_name,omitempty"`
	BuiltinToolsAllow   []string          `json:"builtin_tools_allow,omitempty" yaml:"builtin_tools_allow,omitempty"`
	BuiltinToolsDeny    []string          `json:"builtin_tools_deny,omitempty" yaml:"builtin_tools_deny,omitempty"`
	FewShotExamples     []ExampleTurn     `json:"few_shot_examples,omitempty" yaml:"few_shot_examples,omitempty"`
	ActorState          map[string]string `json:"actor_state,omitempty" yaml:"actor_state,omitempty"`
	StreamPacing        string            `json:"stream_pacing,omitempty" yaml:"stream_pacing,omitempty"`
	CouncilEnabled      bool              `json:"council_enabled,omitempty" yaml:"council_enabled,omitempty"`
	CouncilActors       []string          `json:"council_actors,omitempty" yaml:"council_actors,omitempty"`
	FollowUpActor       string            `json:"follow_up_actor,omitempty" yaml:"follow_up_actor,omitempty"`
	ChatTools           []ChatTool        `json:"chat_tools,omitempty" yaml:"chat_tools,omitempty"`

	InteractionMode    string `json:"interaction_mode,omitempty" yaml:"interaction_mode,omitempty"`       // agent | plan
	ActiveGitBranch    string `json:"active_git_branch,omitempty" yaml:"active_git_branch,omitempty"`
	AgentGoalObjective string `json:"agent_goal_objective,omitempty" yaml:"agent_goal_objective,omitempty"`
	AgentGoalStatus    string `json:"agent_goal_status,omitempty" yaml:"agent_goal_status,omitempty"` // active | complete | cancelled
}

// AgentRuntime holds resolved agent parameters for a single run.
type AgentRuntime struct {
	MaxSteps           int
	ChainOfThought     taijicfg.ChainOfThoughtMode
	ThoughtMaxChars    int
	SystemPromptExtra  string
	Model              string
	MaxTokens          int
	Temperature        float64
	SkillsAllow        []string
	SkillsDeny         []string
	PacksAllow         []string
	MCPServers         []string
	MCPToolsDeny       []string
	BuiltinToolsAllow  []string
	BuiltinToolsDeny   []string
	ChatToolNames      []string
	DecisionsEnabled   bool
	MemoryEnabled      bool
	InteractionMode    string
}

// ResolveProviderID returns the provider id for this session.
func (s *Session) ResolveProviderID(cfg *taijicfg.Config) string {
	if s == nil {
		return ""
	}
	if id := strings.TrimSpace(s.State.Settings.Provider); id != "" {
		return id
	}
	if id := strings.TrimSpace(s.State.Provider); id != "" {
		return id
	}
	if cfg != nil {
		return strings.TrimSpace(cfg.DefaultProvider)
	}
	return ""
}

// AgentRuntimeForRun merges project config with session overrides.
func (s *Session) AgentRuntimeForRun(cfg *taijicfg.Config) AgentRuntime {
	out := AgentRuntime{
		MaxSteps:          25,
		ChainOfThought:    taijicfg.ChainOfThoughtOff,
		DecisionsEnabled:  true,
		MemoryEnabled:     true,
	}
	if cfg != nil {
		out.ChainOfThought = cfg.Agent.ChainOfThought.OrDefault()
		if cfg.Agent.MaxSteps > 0 {
			out.MaxSteps = cfg.Agent.MaxSteps
		}
		if cfg.Agent.ThoughtMaxChars > 0 {
			out.ThoughtMaxChars = cfg.Agent.ThoughtMaxChars
		}
		dec := cfg.ResolvedDecision()
		out.DecisionsEnabled = dec.EnabledOrDefault()
		mem := cfg.ResolvedMemory()
		out.MemoryEnabled = mem.EnabledOrDefault()
	}
	if s == nil {
		return out
	}
	st := s.State.Settings
	if st.MaxSteps != nil && *st.MaxSteps > 0 {
		out.MaxSteps = *st.MaxSteps
	}
	if mode := strings.TrimSpace(st.ChainOfThought); mode != "" {
		out.ChainOfThought = taijicfg.ChainOfThoughtMode(mode).OrDefault()
	}
	if st.ThoughtMaxChars != nil && *st.ThoughtMaxChars > 0 {
		out.ThoughtMaxChars = *st.ThoughtMaxChars
	}
	out.SystemPromptExtra = strings.TrimSpace(st.SystemPromptExtra)
	out.Model = strings.TrimSpace(st.Model)
	if st.MaxTokens != nil && *st.MaxTokens > 0 {
		out.MaxTokens = *st.MaxTokens
	}
	if st.Temperature != nil && *st.Temperature >= 0 {
		out.Temperature = *st.Temperature
	}
	if len(st.SkillsAllow) > 0 {
		out.SkillsAllow = append([]string(nil), st.SkillsAllow...)
	}
	if len(st.SkillsDeny) > 0 {
		out.SkillsDeny = append([]string(nil), st.SkillsDeny...)
	}
	if len(st.PacksAllow) > 0 {
		out.PacksAllow = append([]string(nil), st.PacksAllow...)
	}
	if len(st.MCPServers) > 0 {
		out.MCPServers = append([]string(nil), st.MCPServers...)
	}
	if len(st.MCPToolsDeny) > 0 {
		out.MCPToolsDeny = append([]string(nil), st.MCPToolsDeny...)
	}
	if len(st.BuiltinToolsAllow) > 0 {
		out.BuiltinToolsAllow = append([]string(nil), st.BuiltinToolsAllow...)
	}
	if len(st.BuiltinToolsDeny) > 0 {
		out.BuiltinToolsDeny = append([]string(nil), st.BuiltinToolsDeny...)
	}
	if len(st.ChatTools) > 0 {
		out.ChatToolNames = s.ChatToolNames()
	}
	if st.DecisionsEnabled != nil {
		out.DecisionsEnabled = *st.DecisionsEnabled
	}
	if st.MemoryEnabled != nil {
		out.MemoryEnabled = *st.MemoryEnabled
	}
	out.InteractionMode = strings.TrimSpace(st.InteractionMode)
	if s.GraphActive() {
		for _, name := range GraphBuiltinToolNames() {
			found := false
			for _, allow := range out.BuiltinToolsAllow {
				if strings.EqualFold(strings.TrimSpace(allow), name) {
					found = true
					break
				}
			}
			if !found {
				out.BuiltinToolsAllow = append(out.BuiltinToolsAllow, name)
			}
		}
	}
	return out
}

// UISettingsForDisplay merges UI-related session overrides onto project config.
func (s *Session) UISettingsForDisplay(cfg *taijicfg.Config) taijicfg.UIConfig {
	if cfg == nil {
		return taijicfg.DefaultUIConfig()
	}
	out := cfg.UI
	if s == nil {
		return out
	}
	st := s.State.Settings
	if preset := strings.TrimSpace(st.TurnMetricsPreset); preset != "" {
		out.TurnMetrics.Preset = preset
		out.TurnMetrics.Fields = nil
	}
	if loc := out.LocaleOrDefault(); loc != "" {
		out.ToolLabels.Locale = loc
	}
	return out
}

// PresentationMode returns the chat layout preset for the TUI.
func (s *Session) PresentationMode() string {
	if s == nil {
		return "markdown"
	}
	mode := strings.TrimSpace(s.State.Settings.Presentation)
	if mode == "" {
		mode = strings.TrimSpace(s.State.Settings.Preset)
	}
	if mode == "" {
		return "markdown"
	}
	return mode
}

// SkillAllowed reports whether a skill name may be used in this session.
func (rt AgentRuntime) SkillAllowed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, deny := range rt.SkillsDeny {
		if strings.EqualFold(strings.TrimSpace(deny), name) {
			return false
		}
	}
	if len(rt.SkillsAllow) == 0 {
		return true
	}
	for _, allow := range rt.SkillsAllow {
		if strings.EqualFold(strings.TrimSpace(allow), name) {
			return true
		}
	}
	return false
}

// BuiltinToolAllowed reports whether a built-in tool may be exposed or executed.
func (rt AgentRuntime) BuiltinToolAllowed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	for _, allow := range rt.ChatToolNames {
		if strings.EqualFold(strings.TrimSpace(allow), name) {
			return true
		}
	}
	for _, deny := range rt.BuiltinToolsDeny {
		if strings.EqualFold(strings.TrimSpace(deny), name) {
			return false
		}
	}
	if len(rt.BuiltinToolsAllow) == 0 {
		return true
	}
	for _, allow := range rt.BuiltinToolsAllow {
		if strings.EqualFold(strings.TrimSpace(allow), name) {
			return true
		}
	}
	return false
}

// MCPToolRef builds a stable session deny-list key for a server tool.
func MCPToolRef(server, tool string) string {
	return strings.TrimSpace(server) + ":" + strings.TrimSpace(tool)
}

// ParseMCPToolRef splits a server:tool deny-list entry.
func ParseMCPToolRef(ref string) (server, tool string, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(ref), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	server = strings.TrimSpace(parts[0])
	tool = strings.TrimSpace(parts[1])
	if server == "" || tool == "" {
		return "", "", false
	}
	return server, tool, true
}

// MCPToolAllowed reports whether an MCP tool may run (server enabled and not denied).
func (rt AgentRuntime) MCPToolAllowed(server, tool string) bool {
	if !rt.MCPServerAllowed(server) {
		return false
	}
	tool = strings.TrimSpace(tool)
	if tool == "" {
		return false
	}
	server = strings.TrimSpace(server)
	for _, deny := range rt.MCPToolsDeny {
		ds, dt, ok := ParseMCPToolRef(deny)
		if !ok {
			continue
		}
		if strings.EqualFold(ds, server) && dt == tool {
			return false
		}
	}
	return true
}

// MCPServerAllowed reports whether an MCP server is enabled for this session.
func (rt AgentRuntime) MCPServerAllowed(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if len(rt.MCPServers) == 0 {
		return true
	}
	for _, allow := range rt.MCPServers {
		if strings.EqualFold(strings.TrimSpace(allow), name) {
			return true
		}
	}
	return false
}
