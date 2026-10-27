package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvedPacksDefaults(t *testing.T) {
	var nilCfg *Config
	def := nilCfg.ResolvedPacks()
	if !def.EnabledOrDefault() || !def.KeywordBM25OrDefault() || !def.SemanticOrDefault() {
		t.Fatalf("defaults: %#v", def)
	}
	if def.MaxCharsOrDefault() != 16000 {
		t.Fatalf("max chars=%d", def.MaxCharsOrDefault())
	}
	if def.SemanticTopKOrDefault() != 4 {
		t.Fatalf("topk=%d", def.SemanticTopKOrDefault())
	}
}

func TestDefaultPacksDirs(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	root := t.TempDir()
	dirs := DefaultPacksDirs(root)
	if len(dirs) != 2 {
		t.Fatalf("dirs=%v", dirs)
	}
	wantProject := filepath.Join(RuntimeDir(root), packsSubdir)
	if dirs[0] != wantProject {
		t.Fatalf("project dir=%q", dirs[0])
	}
	if dirs[1] != GlobalPacksDir() {
		t.Fatalf("global dir=%q", dirs[1])
	}
}

func TestGlobalPacksDirUsesXDG(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	if got := GlobalPacksDir(); got != filepath.Join(home, "xdg", "taiji", "packs") {
		t.Fatalf("got=%q", got)
	}
	t.Setenv("XDG_CONFIG_HOME", "")
	fallback := filepath.Join(home, ".config", "taiji", "packs")
	got := GlobalPacksDir()
	if got != fallback {
		t.Fatalf("home fallback=%q", got)
	}
	_ = os.MkdirAll(filepath.Dir(got), 0o755)
}

func TestPacksScoreDefaults(t *testing.T) {
	p := PacksConfig{MinBM25Score: 0.9, MinSemanticScore: 0.8}
	if p.MinBM25ScoreOrDefault() != 0.9 || p.MinSemanticScoreOrDefault() != 0.8 {
		t.Fatal("custom scores")
	}
	zero := PacksConfig{}
	if zero.MinBM25ScoreOrDefault() != 0.45 || zero.MinSemanticScoreOrDefault() != 0.55 {
		t.Fatal("zero defaults")
	}
}
