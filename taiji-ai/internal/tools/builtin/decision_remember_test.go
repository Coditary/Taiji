package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/decisions"
)

func TestDecisionRememberSavesProfile(t *testing.T) {
	dir := t.TempDir()
	tool := DecisionRemember{saveDir: dir, profiles: decisions.NewLoader([]string{dir})}
	res, err := tool.RunWithMetadata(context.Background(), map[string]any{
		"name":        "hire-candidate",
		"description": "Should we hire this candidate?",
		"when":        "User asks whether to hire someone",
		"questions": map[string]any{
			"fit": map[string]any{
				"type":         "noul",
				"instructions": "Is the candidate a good fit?",
			},
			"timing": map[string]any{
				"type":         "choice",
				"instructions": "When to decide?",
				"criteria":     []any{"now", "wait", "reject"},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Data["profile"] != "hire-candidate" {
		t.Fatalf("profile = %#v", res.Data["profile"])
	}
	if _, err := os.Stat(filepath.Join(dir, "hire-candidate", "DECISION.md")); err != nil {
		t.Fatal(err)
	}
}
