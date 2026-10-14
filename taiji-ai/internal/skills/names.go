package skills

import (
	"encoding/json"
	"path/filepath"
	"strings"
)

// skillLookupKey normalizes skill names for comparison.
// Cursor skills use hyphens; models sometimes emit underscores instead.
func skillLookupKey(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	return strings.ReplaceAll(name, "_", "-")
}

// ParseInvokeName reads the skill name from invoke_skill tool arguments.
func ParseInvokeName(args map[string]any) string {
	names := ParseInvokeNames(args)
	if len(names) == 0 {
		return ""
	}
	return names[0]
}

// ParseInvokeNames reads one or more skill names from invoke_skill tool arguments.
func ParseInvokeNames(args map[string]any) []string {
	if args == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" || skillLookupKey(name) == skillLookupKey(InvokeSkillToolName) {
			return
		}
		key := skillLookupKey(name)
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, name)
	}
	for _, key := range []string{"names", "skills", "skill_names"} {
		raw, ok := args[key]
		if !ok {
			continue
		}
		switch v := raw.(type) {
		case []any:
			for _, item := range v {
				if s, ok := item.(string); ok {
					add(s)
				}
			}
		case []string:
			for _, s := range v {
				add(s)
			}
		case string:
			for _, part := range strings.Split(v, ",") {
				add(part)
			}
		}
	}
	for _, key := range []string{"name", "skill", "skill_name"} {
		raw, ok := args[key]
		if !ok {
			continue
		}
		if v, ok := raw.(string); ok {
			add(v)
		}
	}
	if len(out) == 0 {
		if nested := parseInvokeNameNested(args); nested != "" {
			add(nested)
		}
	}
	return out
}

// ParseInvokeNamesFromJSON parses invoke_skill args JSON.
func ParseInvokeNamesFromJSON(argsJSON string) []string {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" {
		return nil
	}
	args := map[string]any{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil
	}
	return ParseInvokeNames(args)
}

// SkillNameFromArgsJSON parses invoke_skill args JSON and returns the skill name.
func SkillNameFromArgsJSON(argsJSON string) string {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" {
		return ""
	}
	names := ParseInvokeNamesFromJSON(argsJSON)
	if len(names) == 0 {
		return ""
	}
	if len(names) == 1 {
		return names[0]
	}
	return strings.Join(names, ", ")
}

func parseInvokeNameNested(args map[string]any) string {
	raw, ok := args["arguments"]
	if !ok {
		return ""
	}
	switch v := raw.(type) {
	case string:
		return SkillNameFromArgsJSON(v)
	case map[string]any:
		names := ParseInvokeNames(v)
		if len(names) == 0 {
			return ""
		}
		if len(names) == 1 {
			return names[0]
		}
		return strings.Join(names, ", ")
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return ""
		}
		return SkillNameFromArgsJSON(string(b))
	}
}

// Find returns the skill with the given invocation name.
// Matches frontmatter name and directory name, treating hyphens and underscores as equivalent.
func Find(all []Skill, name string) (Skill, bool) {
	key := skillLookupKey(name)
	if key == "" {
		return Skill{}, false
	}
	for _, s := range all {
		if skillLookupKey(s.Name) == key {
			return s, true
		}
		if dir := filepath.Base(filepath.Dir(s.Path)); skillLookupKey(dir) == key {
			return s, true
		}
	}
	return Skill{}, false
}
