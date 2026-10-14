package config

import "testing"

func TestResolvedDecisionDefaults(t *testing.T) {
	cfg := defaultConfig("/tmp/taiji")
	got := cfg.ResolvedDecision()
	if !got.EnabledOrDefault() {
		t.Fatal("expected decisions enabled by default")
	}
	if got.Backend != "wuji" {
		t.Fatalf("backend = %q", got.Backend)
	}
	if got.Driver != "laya" {
		t.Fatalf("driver = %q", got.Driver)
	}
}

func TestResolvedDecisionOverride(t *testing.T) {
	cfg := defaultConfig("/tmp/taiji")
	cfg.Decisions = DecisionConfig{
		Backend: "http",
		Driver:  "jev",
		HTTPURL: "http://127.0.0.1:9000",
	}
	got := cfg.ResolvedDecision()
	if got.Backend != "http" {
		t.Fatalf("backend = %q", got.Backend)
	}
	if got.Driver != "jev" {
		t.Fatalf("driver = %q", got.Driver)
	}
	if got.HTTPURL != "http://127.0.0.1:9000" {
		t.Fatalf("http_url = %q", got.HTTPURL)
	}
}
