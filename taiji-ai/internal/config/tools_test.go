package config

import "testing"

func TestResolvedToolsDefaults(t *testing.T) {
	cfg := &Config{}
	tools := cfg.ResolvedTools()
	if !tools.AutoFormatOrDefault() || !tools.AutoVetOrDefault() {
		t.Fatalf("defaults: %#v", tools)
	}
}

func TestToolsAllowLocalFetchDefault(t *testing.T) {
	cfg := &Config{}
	if cfg.ResolvedTools().AllowLocalFetchOrDefault() {
		t.Fatal("expected allow_local_fetch false by default")
	}
	on := true
	cfg = &Config{Tools: ToolsConfig{AllowLocalFetch: &on}}
	if !cfg.ResolvedTools().AllowLocalFetchOrDefault() {
		t.Fatal("expected allow_local_fetch true when set")
	}
}

func TestToolsAutoFormatOverride(t *testing.T) {
	off := false
	cfg := &Config{Tools: ToolsConfig{AutoFormat: &off}}
	if cfg.ResolvedTools().AutoFormatOrDefault() {
		t.Fatal("expected auto_format false")
	}
}
