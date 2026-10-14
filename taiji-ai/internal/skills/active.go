package skills

import (
	"sort"
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

const invokedHeaderPrefix = "## Skill: "

// NameFromInvokedOutput extracts the first skill name from invoke_skill tool output.
func NameFromInvokedOutput(text string) string {
	names := NamesFromInvokedOutput(text)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// NamesFromInvokedOutput extracts all skill names from invoke_skill tool output.
func NamesFromInvokedOutput(text string) []string {
	text = strings.TrimSpace(text)
	if text == "" || strings.HasPrefix(strings.ToLower(text), "error:") {
		return nil
	}
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if !strings.HasPrefix(line, invokedHeaderPrefix) {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, invokedHeaderPrefix))
		if name != "" {
			out = append(out, name)
		}
	}
	return out
}

// ActiveFromMessages returns skill names successfully loaded via invoke_skill in order.
func ActiveFromMessages(messages []provider.Message) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range messages {
		if m.Role != provider.RoleTool || m.Name != InvokeSkillToolName {
			continue
		}
		for _, name := range NamesFromInvokedOutput(m.Content) {
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}
