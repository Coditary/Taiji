package config

import "testing"

func TestThrashingThresholdDefault(t *testing.T) {
	if (AgentConfig{}).ThrashingThresholdOrDefault() != 3 {
		t.Fatal("expected default threshold 3")
	}
	cfg := AgentConfig{ThrashingThreshold: 5}
	if cfg.ThrashingThresholdOrDefault() != 5 {
		t.Fatal("expected custom threshold")
	}
}

func TestCacheFriendlyPruningDefault(t *testing.T) {
	if !(AgentConfig{}).CacheFriendlyPruningOrDefault() {
		t.Fatal("expected cache-friendly pruning default true")
	}
	off := false
	cfg := AgentConfig{CacheFriendlyPruning: &off}
	if cfg.CacheFriendlyPruningOrDefault() {
		t.Fatal("expected false when disabled")
	}
}
