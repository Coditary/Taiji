package skills

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func writeTestSkill(t *testing.T, dir, name, description, body string) {
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

func TestCatalogSearchBM25(t *testing.T) {
	dir := t.TempDir()
	writeTestSkill(t, dir, "generate-image", "Generate images from text prompts", "Use when the user wants an image.")
	writeTestSkill(t, dir, "review", "Review code changes", "Use for pull request review.")

	cfg := &taijicfg.Config{}
	catalog := NewCatalog(NewLoader([]string{dir}), nil, cfg)
	matches, err := catalog.Search(context.Background(), "please generate an image of a dog")
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) == 0 {
		t.Fatal("expected matches")
	}
	if matches[0].Name != "generate-image" {
		t.Fatalf("top match=%q want generate-image", matches[0].Name)
	}
	if matches[0].Fit == "" || matches[0].Fit == "none" {
		t.Fatalf("unexpected fit=%q", matches[0].Fit)
	}
}
