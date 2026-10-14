package memory

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandTermsUsesSynonymGroups(t *testing.T) {
	groups := [][]string{{"name", "heißt", "vorname"}}
	expanded := ExpandTerms([]string{"name"}, groups)
	if len(expanded) < 3 {
		t.Fatalf("expected expanded synonyms, got %v", expanded)
	}
	seen := map[string]bool{}
	for _, term := range expanded {
		seen[term] = true
	}
	for _, want := range []string{"name", "heißt", "vorname"} {
		if !seen[want] {
			t.Fatalf("missing expanded term %q in %v", want, expanded)
		}
	}
}

func TestLoadSynonymGroupsMergesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "memory-synonyms.yaml")
	if err := os.WriteFile(path, []byte("groups:\n  - [tool, werkzeug, cli]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	groups := LoadSynonymGroups(path)
	found := false
	for _, group := range groups {
		for _, term := range group {
			if term == "werkzeug" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected custom synonym group from file")
	}
}
