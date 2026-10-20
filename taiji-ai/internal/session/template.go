package session

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"gopkg.in/yaml.v3"
)

// TemplateSummary is metadata for UI lists.
type TemplateSummary struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

type templateDocument struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Settings    Settings `yaml:"settings"`
}

// TemplateDirs returns search paths (project overrides global overrides bundled).
func TemplateDirs(projectRoot string) []string {
	seen := map[string]struct{}{}
	var dirs []string
	add := func(path string) {
		path = strings.TrimSpace(path)
		if path == "" {
			return
		}
		if _, err := os.Stat(path); err != nil {
			return
		}
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		dirs = append(dirs, path)
	}

	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot != "" {
		add(filepath.Join(taijicfg.RuntimeDir(projectRoot), "templates"))
	}
	globalDir := taijicfg.GlobalConfigDir()
	if globalDir != "" {
		add(filepath.Join(globalDir, "templates"))
	}
	if bundled := bundledTemplatesDir(projectRoot); bundled != "" {
		add(bundled)
	}
	return dirs
}

func bundledTemplatesDir(projectRoot string) string {
	projectRoot = strings.TrimSpace(projectRoot)
	candidates := []string{
		filepath.Join(projectRoot, "taiji-ai", "global", "templates"),
		filepath.Join(projectRoot, "global", "templates"),
	}
	if projectRoot != "" {
		dir := filepath.Dir(projectRoot)
		candidates = append(candidates, filepath.Join(dir, "taiji-ai", "global", "templates"))
	}
	for _, path := range candidates {
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			return path
		}
	}
	return ""
}

// ListChatTemplates returns templates from all template dirs (later dirs override same id).
func ListChatTemplates(projectRoot string) []TemplateSummary {
	byID := map[string]TemplateSummary{}
	for _, dir := range TemplateDirs(projectRoot) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if ent.IsDir() {
				continue
			}
			name := ent.Name()
			if !strings.HasSuffix(strings.ToLower(name), ".yaml") && !strings.HasSuffix(strings.ToLower(name), ".yml") {
				continue
			}
			path := filepath.Join(dir, name)
			doc, err := loadTemplateFile(path)
			if err != nil {
				continue
			}
			id := strings.TrimSpace(doc.ID)
			if id == "" {
				id = strings.TrimSuffix(strings.TrimSuffix(name, ".yaml"), ".yml")
			}
			title := strings.TrimSpace(doc.Name)
			if title == "" {
				title = id
			}
			byID[id] = TemplateSummary{
				ID:          id,
				Name:        title,
				Description: strings.TrimSpace(doc.Description),
				Source:      path,
			}
		}
	}
	out := make([]TemplateSummary, 0, len(byID))
	for _, s := range byID {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func loadTemplateFile(path string) (*templateDocument, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc templateDocument
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	if strings.TrimSpace(doc.ID) == "" {
		base := filepath.Base(path)
		doc.ID = strings.TrimSuffix(strings.TrimSuffix(base, ".yaml"), ".yml")
	}
	return &doc, nil
}

// LoadChatTemplate loads a template by id from search paths.
func LoadChatTemplate(projectRoot, templateID string) (*templateDocument, error) {
	templateID = strings.TrimSpace(templateID)
	if templateID == "" {
		return nil, fmt.Errorf("template id is required")
	}
	var found *templateDocument
	var foundPath string
	for _, dir := range TemplateDirs(projectRoot) {
		for _, ext := range []string{".yaml", ".yml"} {
			path := filepath.Join(dir, templateID+ext)
			doc, err := loadTemplateFile(path)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return nil, err
			}
			if strings.EqualFold(strings.TrimSpace(doc.ID), templateID) {
				found = doc
				foundPath = path
			}
		}
	}
	if found == nil {
		return nil, fmt.Errorf("template %q not found", templateID)
	}
	if foundPath != "" {
		_ = foundPath
	}
	return found, nil
}

// ApplyChatTemplate copies template settings onto the session (replace lists and scalar overrides).
func ApplyChatTemplate(s *Session, projectRoot, templateID string) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	doc, err := LoadChatTemplate(projectRoot, templateID)
	if err != nil {
		return err
	}
	applySettingsReplace(&s.State.Settings, doc.Settings)
	s.State.Settings.TemplateID = strings.TrimSpace(doc.ID)
	if name := strings.TrimSpace(doc.Name); name != "" && strings.TrimSpace(s.State.Name) == "" {
		s.State.Name = name
	}
	if id := strings.TrimSpace(doc.Settings.Provider); id != "" {
		s.State.Provider = id
	}
	return s.Save()
}

func applySettingsReplace(dst *Settings, src Settings) {
	if dst == nil {
		return
	}
	if v := strings.TrimSpace(src.Provider); v != "" {
		dst.Provider = v
	}
	if v := strings.TrimSpace(src.Model); v != "" {
		dst.Model = v
	}
	if src.Temperature != nil {
		v := *src.Temperature
		dst.Temperature = &v
	}
	if src.MaxTokens != nil {
		v := *src.MaxTokens
		dst.MaxTokens = &v
	}
	if src.MaxSteps != nil {
		v := *src.MaxSteps
		dst.MaxSteps = &v
	}
	if v := strings.TrimSpace(src.ChainOfThought); v != "" {
		dst.ChainOfThought = v
	}
	if src.ThoughtMaxChars != nil {
		v := *src.ThoughtMaxChars
		dst.ThoughtMaxChars = &v
	}
	if v := strings.TrimSpace(src.SystemPromptExtra); v != "" {
		dst.SystemPromptExtra = v
	}
	dst.SkillsAllow = append([]string(nil), src.SkillsAllow...)
	dst.SkillsDeny = append([]string(nil), src.SkillsDeny...)
	dst.MCPServers = append([]string(nil), src.MCPServers...)
	if v := strings.TrimSpace(src.Presentation); v != "" {
		dst.Presentation = v
	}
	if v := strings.TrimSpace(src.TurnMetricsPreset); v != "" {
		dst.TurnMetricsPreset = v
	}
	if v := strings.TrimSpace(src.ToolLabelsLocale); v != "" {
		dst.ToolLabelsLocale = v
	}
	if v := strings.TrimSpace(src.Preset); v != "" {
		dst.Preset = v
	}
	if src.DecisionsEnabled != nil {
		v := *src.DecisionsEnabled
		dst.DecisionsEnabled = &v
	}
	if src.MemoryEnabled != nil {
		v := *src.MemoryEnabled
		dst.MemoryEnabled = &v
	}
}
