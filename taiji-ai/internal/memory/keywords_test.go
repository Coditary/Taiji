package memory

import (
	"context"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestEnrichMemoryEntryAddsNameRecallQueries(t *testing.T) {
	catalog := LoadKindCatalog(nil)
	tags, keywords, queries := enrichMemoryEntry(
		catalog,
		KindPreference,
		"Name des Benutzers",
		"Der Benutzer heißt Leonard.",
		"Leonard",
		nil, nil, nil,
	)
	if len(queries) == 0 {
		t.Fatal("expected recall queries for name preference")
	}
	if len(keywords) == 0 {
		t.Fatal("expected keywords for name preference")
	}
	foundQuery := false
	for _, q := range queries {
		if q == "wie ist mein name" {
			foundQuery = true
		}
	}
	if !foundQuery {
		t.Fatalf("expected default name query, got %v", queries)
	}
	if len(tags) == 0 {
		t.Fatal("expected tags")
	}
	_ = tags
}

func TestRecallMatchesNameViaSynonymsAndRecallQueries(t *testing.T) {
	root := t.TempDir()
	cfg := &taijicfg.Config{
		Root:   root,
		Memory: taijicfg.DefaultMemoryConfig(),
	}
	cfg.Memory.GlobalDir = filepath.Join(root, "global-memory")
	svc := NewService(cfg, nil)
	ctx := context.Background()

	_, err := svc.Remember(ctx, RememberRequest{
		Kind:        string(KindPreference),
		Title:       "Name des Benutzers",
		RecallText:  "Der Benutzer heißt Leonard.",
		VerifiedFix: "Leonard",
		Scope:       "local",
	})
	if err != nil {
		t.Fatal(err)
	}

	entries, err := svc.Recall(ctx, RecallRequest{
		Query: "wie ist mein name",
		Kind:  string(KindPreference),
		TopK:  5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected recall via keywords/recall_queries/synonyms")
	}
}
