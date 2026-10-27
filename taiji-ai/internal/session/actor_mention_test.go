package session

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func TestResolveActorMention(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(config.RuntimeDir(root), "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "zwerg.md"), []byte("---\nid: zwerg-schmied\nname: Zwerg\n---\nHi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	id, stripped := ResolveActorMention(root, "@zwerg-schmied Schau dir das an")
	if id != "zwerg-schmied" {
		t.Fatalf("id=%q", id)
	}
	if stripped != "Schau dir das an" {
		t.Fatalf("stripped=%q", stripped)
	}
}
