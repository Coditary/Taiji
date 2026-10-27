package packs

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseDocument splits YAML frontmatter and body (Obsidian-style --- blocks).
func ParseDocument(content string) (NodeFrontmatter, string, error) {
	content = strings.TrimPrefix(content, "\ufeff")
	body := content
	meta := NodeFrontmatter{}
	if !strings.HasPrefix(content, "---\n") {
		return meta, strings.TrimSpace(body), nil
	}
	parts := strings.SplitN(content, "\n---\n", 2)
	if len(parts) != 2 {
		return meta, strings.TrimSpace(body), nil
	}
	yamlBlock := strings.TrimPrefix(parts[0], "---")
	if err := yaml.Unmarshal([]byte(yamlBlock), &meta); err != nil {
		return NodeFrontmatter{}, "", fmt.Errorf("parse frontmatter: %w", err)
	}
	return meta, strings.TrimSpace(parts[1]), nil
}

// ApplyLegacyDefaults maps SKILL.md / DECISION.md fields when injection is unset.
func ApplyLegacyDefaults(meta *NodeFrontmatter, legacy LegacyKind) {
	if meta == nil {
		return
	}
	if legacy == LegacySkill && meta.Injection == "" {
		meta.Injection = ModeManual
	}
	if legacy == LegacyDecision && meta.Injection == "" {
		meta.Injection = ModeManual
	}
	if legacy == LegacyAgentsMD && meta.Injection == "" {
		meta.Injection = ModeAlways
		if meta.Priority == 0 {
			meta.Priority = 100
		}
	}
	if meta.DisableModelInvocation || meta.DisableModelInvocationSnake {
		meta.DisableModelInvocation = true
	}
}
