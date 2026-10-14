package builtin

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
)

func TestInvokeSkillRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "review", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`---
name: review
description: Review code.
---

Check tests.`), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	EnableSkillOrchestration(reg, skills.NewLoader([]string{dir}))

	res, err := reg.Run(context.Background(), skills.InvokeSkillToolName, `{"name":"review"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "Check tests.") {
		t.Fatalf("unexpected text: %q", res.Text)
	}
	if res.Data["skill"] != "review" {
		t.Fatalf("unexpected metadata: %#v", res.Data)
	}
}

func TestInvokeSkillMultipleNames(t *testing.T) {
	dir := t.TempDir()
	for _, spec := range []struct {
		dir  string
		name string
		body string
	}{
		{"hallo", "hallo", "Say hi."},
		{"review", "review", "Check tests."},
	} {
		path := filepath.Join(dir, spec.dir, "SKILL.md")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		content := "---\nname: " + spec.name + "\ndescription: Demo.\n---\n\n" + spec.body
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	reg := tools.NewRegistry()
	EnableSkillOrchestration(reg, skills.NewLoader([]string{dir}))

	res, err := reg.Run(context.Background(), skills.InvokeSkillToolName, `{"names":["hallo","review"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "Say hi.") || !strings.Contains(res.Text, "Check tests.") {
		t.Fatalf("unexpected text: %q", res.Text)
	}
	skillsData, ok := res.Data["skills"].([]string)
	if !ok || len(skillsData) != 2 {
		t.Fatalf("unexpected metadata: %#v", res.Data)
	}
}

func TestInvokeSkillHyphenUnderscoreAlias(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "code-review", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`---
name: code-review
description: Review code.
---

Check style.`), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := tools.NewRegistry()
	EnableSkillOrchestration(reg, skills.NewLoader([]string{dir}))

	res, err := reg.Run(context.Background(), skills.InvokeSkillToolName, `{"name":"code_review"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.Text, "Check style.") {
		t.Fatalf("unexpected text: %q", res.Text)
	}
}
