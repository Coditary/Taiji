package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMemoryReflectOnErrorOrDefault(t *testing.T) {
	off := false
	cfg := &Config{Memory: MemoryConfig{ReflectOnError: &off}}
	if cfg.ResolvedMemory().ReflectOnErrorOrDefault() {
		t.Fatal("expected false")
	}
	if DefaultMemoryConfig().ReflectOnErrorOrDefault() != true {
		t.Fatal("default true")
	}
}

func TestExpandHomeAndSynonymsPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "cfg"))
	if got := expandHome("~/notes"); got != filepath.Join(home, "notes") {
		t.Fatalf("expandHome=%q", got)
	}
	if expandHome("~") != home {
		t.Fatal("tilde alone")
	}
	cfg := &Config{Memory: MemoryConfig{SynonymsFile: "~/syn.yaml"}}
	if got := MemorySynonymsPath(cfg); got != filepath.Join(home, "syn.yaml") {
		t.Fatalf("synonyms=%q", got)
	}
	if MemorySynonymsPath(&Config{}) == "" {
		t.Fatal("expected global synonyms path")
	}
	_ = os.MkdirAll(GlobalConfigDir(), 0o755)
}
