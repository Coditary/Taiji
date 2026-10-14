package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindHyphenUnderscoreAlias(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "find-skills", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`---
name: find-skills
description: Find skills.
---

Body.`), 0o644); err != nil {
		t.Fatal(err)
	}

	all, err := NewLoader([]string{dir}).LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	for _, query := range []string{"find-skills", "find_skills", "FIND_SKILLS"} {
		skill, ok := Find(all, query)
		if !ok {
			t.Fatalf("Find(%q) failed", query)
		}
		if skill.Name != "find-skills" {
			t.Fatalf("Find(%q) name = %q", query, skill.Name)
		}
	}
}

func TestSkillNameFromArgsJSON(t *testing.T) {
	if got := SkillNameFromArgsJSON(`{"name":"review"}`); got != "review" {
		t.Fatalf("name key: got %q", got)
	}
	if got := SkillNameFromArgsJSON(`{"skill":"review"}`); got != "review" {
		t.Fatalf("skill key: got %q", got)
	}
	if got := SkillNameFromArgsJSON(`{"name":"invoke_skill","arguments":{"name":"review"}}`); got != "review" {
		t.Fatalf("nested arguments: got %q", got)
	}
	if got := SkillNameFromArgsJSON(`{"name":"invoke_skill"}`); got != "" {
		t.Fatalf("tool name only: got %q", got)
	}
	if got := SkillNameFromArgsJSON(`{"names":["hallo","review"]}`); got != "hallo, review" {
		t.Fatalf("names array: got %q", got)
	}
}

func TestParseInvokeNames(t *testing.T) {
	names := ParseInvokeNames(map[string]any{"names": []any{"hallo", "review"}})
	if len(names) != 2 || names[0] != "hallo" || names[1] != "review" {
		t.Fatalf("unexpected names: %#v", names)
	}
}
