package config

import "strings"

// ToolLabelsUIConfig customizes activity-log titles for tool calls in the TUI.
// Tool invocation names (e.g. read_file) stay unchanged; only display text is affected.
type ToolLabelsUIConfig struct {
	Locale string            `yaml:"locale,omitempty"`
	Tools  map[string]string `yaml:"tools,omitempty"`
	Scopes map[string]string `yaml:"scopes,omitempty"`
}

// ToolDisplayTitle returns a custom activity title for a tool, or "" to use the default
// "{Scope} · {tool_name}" formatting in the TUI.
func (c *Config) ToolDisplayTitle(toolName string) string {
	if c == nil {
		return ""
	}
	labels := c.ResolvedToolLabels()
	name := strings.TrimSpace(toolName)
	if name == "" {
		return ""
	}
	if label := strings.TrimSpace(labels.Tools[name]); label != "" {
		return label
	}
	return ""
}

// ToolScopeLabel returns the scope prefix (e.g. "Tool", "MCP") for activity headers.
func (c *Config) ToolScopeLabel(scope string) string {
	if c == nil {
		return defaultToolScopeLabel(scope)
	}
	labels := c.ResolvedToolLabels()
	key := strings.ToLower(strings.TrimSpace(scope))
	if label := strings.TrimSpace(labels.Scopes[key]); label != "" {
		return label
	}
	return defaultToolScopeLabel(scope)
}

// ResolvedToolLabels merges locale presets with explicit overrides.
func (c *Config) ResolvedToolLabels() ToolLabelsUIConfig {
	out := ToolLabelsUIConfig{
		Tools:  map[string]string{},
		Scopes: map[string]string{},
	}
	if c == nil {
		return out
	}
	for k, v := range defaultToolScopeLabels() {
		out.Scopes[k] = v
	}
	for k, v := range toolLabelLocalePreset(c.UI.ToolLabels.Locale) {
		out.Tools[k] = v
	}
	for k, v := range c.UI.ToolLabels.Scopes {
		if label := strings.TrimSpace(v); label != "" {
			out.Scopes[strings.ToLower(strings.TrimSpace(k))] = label
		}
	}
	for k, v := range c.UI.ToolLabels.Tools {
		if label := strings.TrimSpace(v); label != "" {
			out.Tools[strings.TrimSpace(k)] = label
		}
	}
	return out
}

func defaultToolScopeLabels() map[string]string {
	return map[string]string{
		"tool":      "Tool",
		"mcp":       "MCP",
		"skill":     "Skill",
		"decision":  "Decision",
		"question":  "Question",
		"thinking":  "Thinking",
	}
}

func defaultToolScopeLabel(scope string) string {
	if label, ok := defaultToolScopeLabels()[strings.ToLower(strings.TrimSpace(scope))]; ok {
		return label
	}
	return "Tool"
}

func toolLabelLocalePreset(locale string) map[string]string {
	switch strings.ToLower(strings.TrimSpace(locale)) {
	case "en", "en-us", "english":
		return englishToolLabels()
	case "de", "de-de", "german", "deutsch":
		return germanToolLabels()
	default:
		return map[string]string{}
	}
}

// englishToolLabels is the default English activity-log naming preset.
func englishToolLabels() map[string]string {
	return map[string]string{
		"list_dir":            "List",
		"read_file":           "Read",
		"write_file":          "Write",
		"edit_file":           "Edit",
		"file_outline":        "Outline",
		"bg_process":          "Background",
		"delegate_task":       "Delegate",
		"workspace_checkpoint": "Checkpoint",
		"todo_plan":           "Plan",
		"batch_edit":          "Batch edit",
		"manage_file":         "Manage file",
		"go_symbol_intel":     "Symbol Intel",
		"glob_search":         "Find files",
		"grep":                "Search",
		"shell":               "Run",
		"run_tests":           "Test",
		"web_fetch":           "Fetch URL",
		"git_inspect":         "Git Inspect",
		"turn_route":          "Route",
		"memory_recall":       "Recall",
		"memory_remember":     "Remember",
		"memory_revise":       "Revise memory",
		"rag_search":          "Search docs",
		"rag_index":           "Index docs",
		"invoke_skill":        "Skill",
		"skill_search":        "Find skill",
		"decide":              "Decide",
		"decision_remember":   "Save decision",
		"ask_user":            "Ask",
		"generate_image":      "Generate image",
		"list_mcp_resources":  "List MCP resources",
		"read_mcp_resource":   "Read MCP resource",
	}
}

func germanToolLabels() map[string]string {
	return map[string]string{
		"list_dir":            "Auflisten",
		"read_file":           "Lesen",
		"write_file":          "Schreiben",
		"edit_file":           "Bearbeiten",
		"file_outline":        "Übersicht",
		"bg_process":          "Hintergrund",
		"delegate_task":       "Delegieren",
		"workspace_checkpoint": "Sicherungspunkt",
		"todo_plan":           "Aufgabenplan",
		"batch_edit":          "Stapel-Edit",
		"manage_file":         "Datei verwalten",
		"go_symbol_intel":     "Symbol-Analyse",
		"glob_search":         "Dateien finden",
		"grep":                "Suchen",
		"shell":               "Ausführen",
		"run_tests":           "Tests ausführen",
		"web_fetch":           "Webseite lesen",
		"git_inspect":         "Git-Historie",
		"turn_route":          "Route",
		"memory_recall":       "Erinnern",
		"memory_remember":     "Merken",
		"memory_revise":       "Memory aktualisieren",
		"rag_search":          "Docs durchsuchen",
		"rag_index":           "Docs indexieren",
		"invoke_skill":        "Skill",
		"skill_search":        "Skill finden",
		"decide":              "Entscheiden",
		"decision_remember":   "Entscheidung speichern",
		"ask_user":            "Fragen",
		"generate_image":      "Bild erzeugen",
		"list_mcp_resources":  "MCP-Ressourcen auflisten",
		"read_mcp_resource":   "MCP-Ressource lesen",
	}
}
