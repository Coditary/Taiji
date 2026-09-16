package rag_test

import (
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/rag"
)

func TestScopeGlobalDefault(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/proj", Wuji: taijicfg.WujiConfig{WujiRoot: "../wuji-ai/wuji-ai"}}
	s := rag.ScopeFromConfig(cfg, rag.ScopeOptions{})
	if s.Mode != rag.ScopeGlobal {
		t.Fatalf("mode=%q", s.Mode)
	}
	if s.WriteRoot() != s.GlobalRoot {
		t.Fatalf("write root should be global")
	}
	if s.WriteStoreDir() != ".wuji" {
		t.Fatalf("write store dir=%q", s.WriteStoreDir())
	}
	if !strings.Contains(s.Describe(), ".wuji") {
		t.Fatalf("describe=%q", s.Describe())
	}
}

func TestScopeLocalMergedRead(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/book", Wuji: taijicfg.WujiConfig{WujiRoot: "/wuji"}}
	s := rag.ScopeFromConfig(cfg, rag.ScopeOptions{Local: true})
	if s.Mode != rag.ScopeLocal {
		t.Fatalf("mode=%q", s.Mode)
	}
	if s.WriteRoot() != "/book" {
		t.Fatalf("write=%q", s.WriteRoot())
	}
	if s.WriteStoreDir() != ".taiji" {
		t.Fatalf("write store dir=%q", s.WriteStoreDir())
	}
	if !strings.Contains(s.Describe(), ".taiji") {
		t.Fatalf("describe=%q", s.Describe())
	}
}

func TestScopeLocalOnly(t *testing.T) {
	cfg := &taijicfg.Config{Root: "/book"}
	s := rag.ScopeFromConfig(cfg, rag.ScopeOptions{LocalOnly: true})
	if s.Mode != rag.ScopeLocalOnly {
		t.Fatalf("mode=%q", s.Mode)
	}
	if !strings.Contains(s.Describe(), "local-only") {
		t.Fatalf("describe=%q", s.Describe())
	}
	if s.WriteStoreDir() != ".taiji" {
		t.Fatalf("write store dir=%q", s.WriteStoreDir())
	}
}
