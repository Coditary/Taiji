package session

import (
	"strings"
	"testing"
)

func TestParseExamples(t *testing.T) {
	body := `# Persona
Be concise.

<example>
{{user}}: Hello there
{{actor}}: "Hey."
</example>
`
	examples := ParseExamples(body)
	if len(examples) != 2 {
		t.Fatalf("expected 2 turns, got %d", len(examples))
	}
	if examples[0].Role != "user" || examples[1].Role != "assistant" {
		t.Fatalf("roles: %#v", examples)
	}
	persona := StripExamples(body)
	if strings.Contains(persona, "<example>") {
		t.Fatalf("strip left example tag: %q", persona)
	}
}
