package packs

import (
	"os"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestIndexCollectionDefault(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, "packs")}}}
	idx := NewIndex(NewLoader(cfg.Packs.Dirs), nil, cfg, root)
	if idx.Collection() != "context-packs" {
		t.Fatalf("collection=%q", idx.Collection())
	}
}

func TestIndexReloadAndSummarize(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, ".taiji", "packs", "lore")
	_ = os.MkdirAll(packRoot, 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "a.md"), []byte("---\ninjection: always\n---\nBody"), 0o644)
	_ = os.WriteFile(filepath.Join(packRoot, "b.md"), []byte("---\ninjection: always\n---\nMore"), 0o644)

	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	idx := NewIndex(NewLoader(cfg.Packs.Dirs), nil, cfg, root)
	if err := idx.Reload(); err != nil {
		t.Fatal(err)
	}
	nodes := idx.Nodes()
	if len(nodes) < 2 {
		t.Fatalf("nodes=%d", len(nodes))
	}
	sums := Summarize(nodes)
	if len(sums) != 1 || sums[0].NodeCount < 2 {
		t.Fatalf("sums=%v", sums)
	}
	mentions := ListMentions(nodes, nil)
	if len(mentions) < 2 {
		t.Fatalf("mentions=%d", len(mentions))
	}
}

func TestNewIndexDisabledWhenPacksOff(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Packs: taijicfg.PacksConfig{Enabled: boolPtr(false)}}
	if NewIndex(NewLoader(nil), nil, cfg, cfg.Root) != nil {
		t.Fatal("expected nil index when packs disabled")
	}
}

func boolPtr(v bool) *bool { return &v }
