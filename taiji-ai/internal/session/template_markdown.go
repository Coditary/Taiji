package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/coditary/taiji/internal/packs"
	"gopkg.in/yaml.v3"
)

type templateMarkdownFM struct {
	ID          string         `yaml:"id"`
	Name        string         `yaml:"name"`
	Description string         `yaml:"description"`
	Avatar      string         `yaml:"avatar"`
	Role        string         `yaml:"role"`
	Settings    Settings       `yaml:"settings"`
	UI          templateUI     `yaml:"ui"`
	Tools        templateTools  `yaml:"tools"`
	StateSchema  map[string]any `yaml:"state_schema"`
	DefaultPacks  []string `yaml:"default_packs"`
	FollowUpActor string   `yaml:"follow_up_actor"`
}

type templateUI struct {
	ThemeColor   string `yaml:"theme_color"`
	Layout       string `yaml:"layout"`
	StreamPacing string `yaml:"stream_pacing"`
}

type templateTools struct {
	Allow []string `yaml:"allow"`
	Deny  []string `yaml:"deny"`
}

func loadMarkdownTemplateFile(path string) (*templateDocument, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	meta, body, err := packs.ParseDocument(string(raw))
	if err != nil {
		return nil, err
	}
	_ = meta

	yamlBlock := strings.TrimPrefix(string(raw), "\ufeff")
	if strings.HasPrefix(yamlBlock, "---\n") {
		parts := strings.SplitN(yamlBlock, "\n---\n", 2)
		if len(parts) == 2 {
			yamlBlock = strings.TrimPrefix(parts[0], "---")
		}
	}

	var fm templateMarkdownFM
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, fmt.Errorf("parse template frontmatter: %w", err)
	}
	var flat Settings
	_ = yaml.Unmarshal([]byte(yamlBlock), &flat)

	doc := &templateDocument{
		ID:          strings.TrimSpace(fm.ID),
		Name:        strings.TrimSpace(fm.Name),
		Description: strings.TrimSpace(fm.Description),
		Settings:    mergeTemplateSettings(fm.Settings, flat),
	}
	if doc.ID == "" {
		base := filepath.Base(path)
		doc.ID = strings.TrimSuffix(strings.TrimSuffix(base, ".md"), ".MD")
	}
	if doc.Name == "" {
		doc.Name = doc.ID
	}
	if v := strings.TrimSpace(fm.Avatar); v != "" {
		doc.Settings.Avatar = v
	}
	if v := strings.TrimSpace(fm.UI.ThemeColor); v != "" {
		doc.Settings.ThemeColor = v
	}
	if v := strings.TrimSpace(fm.UI.Layout); v != "" && strings.TrimSpace(doc.Settings.Presentation) == "" {
		doc.Settings.Presentation = v
	}
	if v := strings.TrimSpace(fm.UI.StreamPacing); v != "" {
		doc.Settings.StreamPacing = v
	}
	if len(fm.Tools.Allow) > 0 {
		doc.Settings.BuiltinToolsAllow = append([]string(nil), fm.Tools.Allow...)
	}
	if len(fm.Tools.Deny) > 0 {
		doc.Settings.BuiltinToolsDeny = append([]string(nil), fm.Tools.Deny...)
	}
	if len(fm.DefaultPacks) > 0 {
		doc.Settings.DefaultPacks = append([]string(nil), fm.DefaultPacks...)
	}
	if v := strings.TrimSpace(fm.FollowUpActor); v != "" {
		doc.Settings.FollowUpActor = v
	}
	if role := strings.TrimSpace(fm.Role); role != "" && strings.TrimSpace(doc.Settings.TemplateDisplayName) == "" {
		doc.Settings.TemplateDisplayName = role
	} else if strings.TrimSpace(doc.Settings.TemplateDisplayName) == "" {
		doc.Settings.TemplateDisplayName = doc.Name
	}

	examples := ParseExamples(body)
	persona := StripExamples(body)
	if strings.TrimSpace(doc.Settings.SystemPromptExtra) == "" {
		doc.Settings.SystemPromptExtra = strings.TrimSpace(persona)
	} else if persona != "" {
		doc.Settings.SystemPromptExtra = strings.TrimSpace(doc.Settings.SystemPromptExtra) + "\n\n" + persona
	}
	doc.FewShotExamples = examples
	doc.StateSchema = fm.StateSchema
	return doc, nil
}

func mergeTemplateSettings(nested, flat Settings) Settings {
	out := flat
	if v := strings.TrimSpace(nested.Provider); v != "" {
		out.Provider = v
	}
	if v := strings.TrimSpace(nested.Model); v != "" {
		out.Model = v
	}
	if nested.Temperature != nil {
		out.Temperature = nested.Temperature
	}
	if nested.MaxTokens != nil {
		out.MaxTokens = nested.MaxTokens
	}
	if nested.MaxSteps != nil {
		out.MaxSteps = nested.MaxSteps
	}
	if v := strings.TrimSpace(nested.ChainOfThought); v != "" {
		out.ChainOfThought = v
	}
	if nested.ThoughtMaxChars != nil {
		out.ThoughtMaxChars = nested.ThoughtMaxChars
	}
	if v := strings.TrimSpace(nested.SystemPromptExtra); v != "" {
		out.SystemPromptExtra = v
	}
	if len(nested.SkillsAllow) > 0 {
		out.SkillsAllow = append([]string(nil), nested.SkillsAllow...)
	}
	if len(nested.SkillsDeny) > 0 {
		out.SkillsDeny = append([]string(nil), nested.SkillsDeny...)
	}
	if len(nested.PacksAllow) > 0 {
		out.PacksAllow = append([]string(nil), nested.PacksAllow...)
	}
	if len(nested.DefaultPacks) > 0 {
		out.DefaultPacks = append([]string(nil), nested.DefaultPacks...)
	}
	if len(out.DefaultPacks) == 0 && len(flat.DefaultPacks) > 0 {
		out.DefaultPacks = append([]string(nil), flat.DefaultPacks...)
	}
	if len(nested.MCPServers) > 0 {
		out.MCPServers = append([]string(nil), nested.MCPServers...)
	}
	if len(nested.MCPToolsDeny) > 0 {
		out.MCPToolsDeny = append([]string(nil), nested.MCPToolsDeny...)
	}
	if v := strings.TrimSpace(nested.Presentation); v != "" {
		out.Presentation = v
	}
	if v := strings.TrimSpace(nested.TurnMetricsPreset); v != "" {
		out.TurnMetricsPreset = v
	}
	if v := strings.TrimSpace(nested.ToolLabelsLocale); v != "" {
		out.ToolLabelsLocale = v
	}
	if v := strings.TrimSpace(nested.Preset); v != "" {
		out.Preset = v
	}
	if nested.DecisionsEnabled != nil {
		out.DecisionsEnabled = nested.DecisionsEnabled
	}
	if nested.MemoryEnabled != nil {
		out.MemoryEnabled = nested.MemoryEnabled
	}
	if nested.ContextWindow != nil {
		out.ContextWindow = nested.ContextWindow
	}
	if len(nested.BuiltinToolsAllow) > 0 {
		out.BuiltinToolsAllow = append([]string(nil), nested.BuiltinToolsAllow...)
	}
	if len(nested.BuiltinToolsDeny) > 0 {
		out.BuiltinToolsDeny = append([]string(nil), nested.BuiltinToolsDeny...)
	}
	if len(nested.DefaultPacks) > 0 {
		out.DefaultPacks = append([]string(nil), nested.DefaultPacks...)
	}
	if v := strings.TrimSpace(nested.Avatar); v != "" {
		out.Avatar = v
	}
	if v := strings.TrimSpace(nested.ThemeColor); v != "" {
		out.ThemeColor = v
	}
	if v := strings.TrimSpace(nested.TemplateDisplayName); v != "" {
		out.TemplateDisplayName = v
	}
	return out
}

func actorStateFromSchema(schema map[string]any) map[string]string {
	if len(schema) == 0 {
		return nil
	}
	out := make(map[string]string, len(schema))
	for k, v := range schema {
		key := strings.TrimSpace(k)
		if key == "" {
			continue
		}
		out[key] = schemaValueString(v)
	}
	return out
}

func schemaValueString(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(x)
	case bool:
		if x {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case float64:
		return strconv.FormatFloat(x, 'f', -1, 64)
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}
