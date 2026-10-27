package rag_test

import (
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

func TestShouldAutoIndexPath(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"docs/readme.md", true},
		{"src/main.go", true},
		{".wuji/rag/default/x", false},
		{"node_modules/pkg/index.js", false},
		{".taiji/rag/default/x", false},
		{".taiji/raggo/chromem.db", false},
		{"image.png", false},
		{"Makefile", false},
		{".config/taiji/packs/demo/node.md", true},
	}
	for _, tc := range tests {
		if got := rag.ShouldAutoIndexPath(tc.path); got != tc.want {
			t.Fatalf("ShouldAutoIndexPath(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestSystemPromptAppendix(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/book", Wuji: taijicfg.WujiConfig{WujiRoot: "/wuji"}}
	svc, err := rag.NewService(cfg, rag.Options{Local: true})
	if err != nil {
		t.Fatal(err)
	}
	appendix := rag.SystemPromptAppendix(svc)
	if appendix == "" {
		t.Fatal("expected non-empty appendix")
	}
	for _, needle := range []string{"rag_search", "rag_index", "Do not set store_root", svc.DefaultCollection} {
		if !strings.Contains(appendix, needle) {
			t.Fatalf("appendix missing %q: %s", needle, appendix)
		}
	}
}
