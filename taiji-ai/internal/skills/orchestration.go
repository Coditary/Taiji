package skills

import (
	"fmt"
	"sort"
	"strings"
)

const InvokeSkillToolName = "invoke_skill"

// SystemPromptAppendix lists invokable skills for the agent system prompt.
func SystemPromptAppendix(loader *Loader) (string, error) {
	if loader == nil {
		return "", nil
	}
	all, err := loader.LoadAll()
	if err != nil {
		return "", err
	}
	return BuildCatalogAppendix(Invokable(all)), nil
}

// Invokable returns skills the agent may load via invoke_skill.
func Invokable(all []Skill) []Skill {
	out := make([]Skill, 0, len(all))
	for _, s := range all {
		if s.DisableModelInvocation {
			continue
		}
		out = append(out, s)
	}
	return out
}

// BuildCatalogAppendix formats name and description for each invokable skill.
func BuildCatalogAppendix(invokable []Skill) string {
	if len(invokable) == 0 {
		return ""
	}
	sorted := append([]Skill(nil), invokable...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	var b strings.Builder
	b.WriteString("## Agent skills\n\n")
	b.WriteString("Specialized workflows live in skills. Each entry shows the invocation name and when to use it.\n")
	b.WriteString(fmt.Sprintf("Call %s to load full instructions before acting.\n", InvokeSkillToolName))
	b.WriteString("Multiple skills: use {\"names\": [\"skill-a\", \"skill-b\"]} in one call, or parallel invoke_skill calls in the same turn.\n")
	b.WriteString("After any tool calls, stop and wait for all results. Never answer the user in the same turn as invoke_skill.\n")
	b.WriteString("Independent read/search/shell tool_calls may also run in parallel in the same turn.\n")
	b.WriteString("Never paste loaded skill text into the user-facing reply — follow the instructions internally.\n\n")
	for _, s := range sorted {
		line := "- " + s.Name
		if desc := strings.TrimSpace(s.Description); desc != "" {
			line += ": " + desc
		}
		b.WriteString(line + "\n")
	}
	return strings.TrimSpace(b.String())
}

// FormatInvoked returns the tool output for a loaded skill.
func FormatInvoked(s Skill) string {
	return strings.TrimSpace(fmt.Sprintf(
		"## Skill: %s\n\n%s\n\nUse these instructions internally for the current task. Do not quote or repeat this skill text to the user. Load any other needed skills with %s before answering.",
		s.Name, s.Body, InvokeSkillToolName,
	))
}
