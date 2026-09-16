package provider

import (
	"strings"
	"testing"
)

func TestValidateUnixSocketPathTooLong(t *testing.T) {
	long := strings.Repeat("a", 120)
	err := ValidateUnixSocketPath(long)
	if err == nil {
		t.Fatal("expected error for long path")
	}
	if !strings.Contains(err.Error(), "too long") {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestResolveWujiRootUsesEnv(t *testing.T) {
	t.Setenv("WUJI_ROOT", "/opt/wuji")
	got := resolveWujiRoot("../ignored", "/proj")
	if got != "/opt/wuji" {
		t.Fatalf("got %q", got)
	}
}
