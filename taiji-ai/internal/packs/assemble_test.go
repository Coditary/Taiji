package packs

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestAssembleSkipsAgentsMDNode(t *testing.T) {
	root := t.TempDir()
	body := strings.Repeat("Project rule line.\n", 2000)
	_ = os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte(body), 0o644)

	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	loader := NewLoader(cfg.Packs.Dirs)
	idx := NewIndex(loader, nil, cfg, root)
	if err := idx.ReloadAt(root); err != nil {
		t.Fatal(err)
	}
	if len(idx.Nodes()) == 0 {
		t.Fatal("expected AGENTS.md node in index")
	}
	res, err := idx.Assemble(context.Background(), AssembleRequest{Goal: "hello"})
	if err != nil {
		t.Fatal(err)
	}
	if contains(res.SystemAppendix, "Project rule line.") {
		t.Fatalf("AGENTS.md must not go through pack assembly: %q", res.SystemAppendix)
	}
}

func TestAssembleAlwaysAndKeyword(t *testing.T) {
	root := t.TempDir()
	packRoot := filepath.Join(root, ".taiji", "packs", "lore")
	_ = os.MkdirAll(packRoot, 0o755)
	_ = os.WriteFile(filepath.Join(packRoot, "guild.md"), []byte(`---
name: schattengilde
injection: keyword
keywords: [diebe]
---
Secret guild lore.
`), 0o644)
	_ = os.WriteFile(filepath.Join(packRoot, "rules.md"), []byte(`---
injection: always
priority: 5
---
Always rules.
`), 0o644)

	cfg := &taijicfg.Config{Root: root, Packs: taijicfg.PacksConfig{Dirs: []string{filepath.Join(root, ".taiji", "packs")}}}
	loader := NewLoader(cfg.Packs.Dirs)
	idx := NewIndex(loader, nil, cfg, root)
	if err := idx.Reload(); err != nil {
		t.Fatal(err)
	}
	res, err := idx.Assemble(context.Background(), AssembleRequest{Goal: "Die Diebe kamen in die Stadt."})
	if err != nil {
		t.Fatal(err)
	}
	if res.SystemAppendix == "" || !contains(res.SystemAppendix, "Always rules") {
		t.Fatalf("missing always: %q", res.SystemAppendix)
	}
	if !contains(res.SystemAppendix, "Secret guild lore") {
		t.Fatalf("missing keyword inject: %q", res.SystemAppendix)
	}
}
