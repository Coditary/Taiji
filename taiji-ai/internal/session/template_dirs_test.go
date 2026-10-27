package session

import (
	"os"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestTemplateDirsAndListChatTemplates(t *testing.T) {
	root := t.TempDir()
	tmplDir := filepath.Join(taijicfg.RuntimeDir(root), "templates")
	if err := os.MkdirAll(tmplDir, 0o755); err != nil {
		t.Fatal(err)
	}
	yaml := `id: test-tmpl
name: Test Template
description: For tests
settings:
  presentation: story
`
	if err := os.WriteFile(filepath.Join(tmplDir, "test-tmpl.yaml"), []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	dirs := TemplateDirs(root)
	if len(dirs) == 0 {
		t.Fatal("expected template dirs")
	}
	list := ListChatTemplates(root)
	var found bool
	for _, item := range list {
		if item.ID == "test-tmpl" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("test-tmpl missing from list=%v", list)
	}
}

func TestApplyChatTemplateMissing(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "s1", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := ApplyChatTemplate(s, root, "missing"); err == nil {
		t.Fatal("expected error")
	}
}
