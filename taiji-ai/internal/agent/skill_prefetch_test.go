package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/skills"
)

func TestRunSkillPrefetchEmitsSearch(t *testing.T) {
	dir := t.TempDir()
	writeSkillForTest(t, dir, "generate-image", "Generate images from text prompts", "image generation workflow")

	catalog := skills.NewCatalog(skills.NewLoader([]string{dir}), nil, nil)
	var kinds []string
	matches := runSkillPrefetch(context.Background(), RunRequest{
		SkillsCatalog: catalog,
	}, "generate image of a cat", func(ev events.Event) {
		kinds = append(kinds, ev.ToolName)
	})
	if len(matches) == 0 {
		t.Fatal("expected skill matches")
	}
	if matches[0].Name != "generate-image" {
		t.Fatalf("top match=%q", matches[0].Name)
	}
	if len(kinds) < 2 || kinds[0] != skillSearchToolName || kinds[1] != skillSearchToolName {
		t.Fatalf("expected skill_search events, got %v", kinds)
	}
}

func writeSkillForTest(t *testing.T, dir, name, description, body string) {
	t.Helper()
	skillDir := filepath.Join(dir, name)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := "---\nname: " + name + "\ndescription: " + description + "\n---\n" + body
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
