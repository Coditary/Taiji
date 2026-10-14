package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSystemPromptAppendix(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "public-skill", `---
name: public-skill
description: Visible in catalog.
---

Public body.`)
	writeSkill(t, dir, "hidden-skill", `---
name: hidden-skill
description: Hidden from catalog.
disable-model-invocation: true
---

Hidden body.`)

	loader := NewLoader([]string{dir})
	appendix, err := SystemPromptAppendix(loader)
	if err != nil {
		t.Fatal(err)
	}
	for _, needle := range []string{
		"## Agent skills",
		InvokeSkillToolName,
		"public-skill: Visible in catalog.",
	} {
		if !strings.Contains(appendix, needle) {
			t.Fatalf("appendix missing %q:\n%s", needle, appendix)
		}
	}
	if strings.Contains(appendix, "hidden-skill") {
		t.Fatalf("hidden skill should not appear in catalog:\n%s", appendix)
	}
}

func TestInvokeSkillTool(t *testing.T) {
	dir := t.TempDir()
	writeSkill(t, dir, "debugging", `---
name: debugging
description: Debug workflow.
---

Step one.`)

	loader := NewLoader([]string{dir})
	all, err := loader.LoadAll()
	if err != nil {
		t.Fatal(err)
	}
	skill, ok := Find(all, "debugging")
	if !ok {
		t.Fatal("skill not found")
	}
	out := FormatInvoked(skill)
	if !strings.Contains(out, "## Skill: debugging") || !strings.Contains(out, "Step one.") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func writeSkill(t *testing.T, root, dirName, content string) {
	path := filepath.Join(root, dirName, "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
