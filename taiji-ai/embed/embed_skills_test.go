package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/skills"
)

func TestFormatSkillsList(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "review")
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: review\ndescription: Code review workflow.\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	h := &runtimeHandle{
		rt: &runtime.Runtime{
			Skills: skills.NewLoader([]string{dir}),
		},
	}
	out := formatSkillsList(h)
	if !strings.Contains(out, "review") || !strings.Contains(out, "Code review workflow") {
		t.Fatalf("unexpected output: %q", out)
	}
}
