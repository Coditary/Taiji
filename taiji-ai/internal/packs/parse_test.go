package packs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseDocumentYAML(t *testing.T) {
	raw := `---
name: schattengilde
injection: keyword
keywords: [diebe, guild]
priority: 10
---
# Lore
Die Schattengilde regiert die Slums.
`
	meta, body, err := ParseDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	if meta.Name != "schattengilde" || meta.Injection != ModeKeyword {
		t.Fatalf("meta: %+v", meta)
	}
	if len(meta.Keywords) != 2 {
		t.Fatalf("keywords: %v", meta.Keywords)
	}
	if body == "" || !contains(body, "Schattengilde") {
		t.Fatalf("body: %q", body)
	}
}

func TestLegacySkillDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo", "SKILL.md")
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, []byte("---\nname: demo\ndescription: test\n---\nBody\n"), 0o644)
	node, err := LoadLegacySkillNode(path, "skills")
	if err != nil {
		t.Fatal(err)
	}
	if node.EffectiveInjection() != ModeManual {
		t.Fatalf("injection %q", node.EffectiveInjection())
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
