package config

import "testing"

func TestPruneStaleObservationsDefault(t *testing.T) {
	cfg := AgentConfig{}
	if !cfg.PruneStaleObservationsOrDefault() {
		t.Fatal("expected default true")
	}
	off := false
	cfg.PruneStaleObservations = &off
	if cfg.PruneStaleObservationsOrDefault() {
		t.Fatal("expected false")
	}
}
