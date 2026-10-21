package config

import (
	"strings"

	"github.com/coditary/taiji/internal/uilocale"
)

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
	locale := toolLabelsLocale(c)
	for k, v := range uilocale.PrefixMap(locale, "tools", "scopes.") {
		out.Scopes[k] = v
	}
	for k, v := range uilocale.PrefixMap(locale, "tools", "names.") {
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

func toolLabelsLocale(c *Config) string {
	if c == nil {
		return "en"
	}
	locale := strings.TrimSpace(c.UI.ToolLabels.Locale)
	if locale == "" {
		locale = c.UI.LocaleOrDefault()
	}
	return NormalizeUILocale(locale)
}

func defaultToolScopeLabel(scope string) string {
	key := strings.ToLower(strings.TrimSpace(scope))
	if label := uilocale.Lookup("en", "tools", "scopes."+key); label != "" {
		return label
	}
	return "Tool"
}
