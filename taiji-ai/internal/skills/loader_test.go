package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFileFrontmatter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "demo-skill", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `---
name: demo-skill
description: Demo skill for tests.
disable-model-invocation: true
---

# Demo

Follow these steps.`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	skill, err := LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if skill.Name != "demo-skill" {
		t.Fatalf("name: got %q", skill.Name)
	}
	if skill.Description != "Demo skill for tests." {
		t.Fatalf("description: got %q", skill.Description)
	}
	if !skill.DisableModelInvocation {
		t.Fatal("expected disable-model-invocation")
	}
	if skill.Body != "# Demo\n\nFollow these steps." {
		t.Fatalf("body: got %q", skill.Body)
	}
}

func TestBuildSystemAppendix(t *testing.T) {
	all := []Skill{
		{Name: "a", Body: "body a"},
		{Name: "b", Body: "body b"},
	}
	out := BuildSystemAppendix(all, []string{"b"})
	if out != "## Skill: b\n\nbody b" {
		t.Fatalf("unexpected appendix: %q", out)
	}
}
