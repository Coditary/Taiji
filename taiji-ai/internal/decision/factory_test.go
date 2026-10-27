package decision

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestNewServiceValidation(t *testing.T) {
	if _, err := NewService(nil); err == nil {
		t.Fatal("expected error for nil config")
	}
	disabled := false
	cfg := &taijicfg.Config{Decisions: taijicfg.DecisionConfig{Enabled: &disabled}}
	if _, err := NewService(cfg); err != ErrDisabled {
		t.Fatalf("err=%v", err)
	}
	cfg.Decisions = taijicfg.DecisionConfig{Enabled: boolPtr(true), Backend: "nope"}
	if _, err := NewService(cfg); err == nil {
		t.Fatal("expected unsupported backend error")
	}
}

func TestNewServiceHTTPBackend(t *testing.T) {
	cfg := &taijicfg.Config{
		Root: t.TempDir(),
		Decisions: taijicfg.DecisionConfig{
			Enabled: boolPtr(true),
			Backend: "http",
			HTTPURL: "http://127.0.0.1:9",
			Driver:  "laya",
		},
	}
	svc, err := NewService(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if svc.BackendName != "http" || svc.DescribeBackend() != "http/laya" {
		t.Fatalf("svc=%+v", svc)
	}
	if err := svc.Close(); err != nil {
		t.Fatal(err)
	}
}

func boolPtr(b bool) *bool { return &b }
