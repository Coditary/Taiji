package config

import "testing"

func TestResolvedMemoryDefaults(t *testing.T) {
	cfg := &Config{}
	mem := cfg.ResolvedMemory()
	if !mem.EnabledOrDefault() {
		t.Fatal("expected memory enabled by default")
	}
	if mem.RAGCollection != "memory" {
		t.Fatalf("collection=%q", mem.RAGCollection)
	}
	if mem.MinRecallScore <= 0 {
		t.Fatalf("min score=%v", mem.MinRecallScore)
	}
	if !mem.AutoRecallOnErrorOrDefault() {
		t.Fatal("expected auto recall on error")
	}
	if mem.PrefetchOnTurnOrDefault() {
		t.Fatal("prefetch on turn should default to false")
	}
}

func TestMemoryPrefetchOnTurnConfig(t *testing.T) {
	on := true
	cfg := &Config{Memory: MemoryConfig{PrefetchOnTurn: &on}}
	if !cfg.ResolvedMemory().PrefetchOnTurnOrDefault() {
		t.Fatal("expected prefetch enabled")
	}
}

func TestProjectMemoryDir(t *testing.T) {
	got := ProjectMemoryDir("/tmp/proj")
	want := "/tmp/proj/.taiji/memory"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
