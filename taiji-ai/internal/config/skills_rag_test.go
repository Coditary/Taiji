package config

import "testing"

func TestResolvedSkillsRAGMerge(t *testing.T) {
	disabled := false
	cfg := &Config{
		Skills: SkillsConfig{
			RAG: SkillsRAGConfig{
				Enabled:    &disabled,
				Collection: " custom ",
				TopK:       9,
				MinScore:   0.7,
			},
		},
	}
	got := cfg.ResolvedSkillsRAG()
	if got.EnabledOrDefault() || got.PrefetchOrDefault() != true {
		t.Fatalf("enabled/prefetch = %v %v", got.EnabledOrDefault(), got.PrefetchOrDefault())
	}
	if got.Collection != "custom" || got.TopK != 9 || got.MinScore != 0.7 {
		t.Fatalf("rag = %+v", got)
	}
	if (*Config)(nil).ResolvedSkillsRAG().TopK != DefaultSkillsRAGConfig().TopK {
		t.Fatal("nil config should use defaults")
	}
}
