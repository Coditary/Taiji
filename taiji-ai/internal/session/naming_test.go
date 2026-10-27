package session

import (
	"context"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func TestGenerateNameUsesDummyEcho(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:            root,
		DefaultProvider: "dummy",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy": {Type: "dummy", Model: "alpha"},
		},
	}
	reg, err := provider.NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	name, err := GenerateName(context.Background(), reg, "dummy", "Fix the failing unit tests")
	if err != nil {
		t.Fatal(err)
	}
	if name == "" {
		t.Fatal("expected non-empty name")
	}
}

func TestFallbackNameTruncates(t *testing.T) {
	long := "one two three four five six seven eight"
	got := fallbackName(long)
	if len(strings.Fields(got)) > 6 {
		t.Fatalf("fallback=%q", got)
	}
}
