package session

import (
	"regexp"
	"strings"
)

// ExampleTurn is one few-shot dialog line for ephemeral prompt injection.
type ExampleTurn struct {
	Role    string `json:"role" yaml:"role"`
	Content string `json:"content" yaml:"content"`
}

var (
	reExampleBlock = regexp.MustCompile(`(?is)<example>\s*(.*?)\s*</example>`)
	reExampleLine  = regexp.MustCompile(`^\s*\{\{(user|actor|char|assistant)\}\}\s*:\s*(.*)$`)
)

// StripExamples removes <example> blocks and returns personality markdown.
func StripExamples(body string) string {
	return strings.TrimSpace(reExampleBlock.ReplaceAllString(body, ""))
}

// ParseExamples extracts dialog turns from markdown <example> blocks.
func ParseExamples(body string) []ExampleTurn {
	var out []ExampleTurn
	for _, m := range reExampleBlock.FindAllStringSubmatch(body, -1) {
		if len(m) < 2 {
			continue
		}
		for _, line := range strings.Split(m[1], "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			sub := reExampleLine.FindStringSubmatch(line)
			if len(sub) != 3 {
				continue
			}
			role := strings.ToLower(strings.TrimSpace(sub[1]))
			switch role {
			case "user":
				role = "user"
			case "actor", "char", "assistant":
				role = "assistant"
			default:
				continue
			}
			content := strings.TrimSpace(sub[2])
			if content == "" {
				continue
			}
			if len(content) >= 2 {
				if (content[0] == '"' && content[len(content)-1] == '"') ||
					(content[0] == '\'' && content[len(content)-1] == '\'') {
					content = content[1 : len(content)-1]
				}
			}
			out = append(out, ExampleTurn{Role: role, Content: content})
		}
	}
	return out
}
