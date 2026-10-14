package memory

import (
	"strings"
	"testing"
)

func sampleNameEntry() Entry {
	return Entry{
		ID:         "name-des-benutzers",
		Kind:       KindPreference,
		Title:      "Name des Benutzers",
		RecallText: "Der Benutzer heißt Leonard.",
		Scope:      "local",
	}
}

func TestBM25SearchNameQueries(t *testing.T) {
	entries := []Entry{sampleNameEntry()}
	cases := []struct {
		query    string
		wantHit  bool
		wantID   string
	}{
		{"name", true, "name-des-benutzers"},
		{"Leonard", true, "name-des-benutzers"},
		{"Wie heiße ich?", true, "name-des-benutzers"},
		{"Name des Benutzers", true, "name-des-benutzers"},
		{"compile error", false, ""},
	}
	for _, tc := range cases {
		hits := searchBM25(entries, tc.query, string(KindPreference), defaultSynonymGroups)
		if tc.wantHit {
			if len(hits) == 0 {
				t.Fatalf("query %q: expected hit", tc.query)
			}
			if hits[0].ID != tc.wantID {
				t.Fatalf("query %q: got id=%q want=%q", tc.query, hits[0].ID, tc.wantID)
			}
			if hits[0].MatchSource != "bm25" {
				t.Fatalf("query %q: match source=%q", tc.query, hits[0].MatchSource)
			}
			if hits[0].Score <= 0 {
				t.Fatalf("query %q: expected positive score, got %f", tc.query, hits[0].Score)
			}
			continue
		}
		if len(hits) != 0 {
			t.Fatalf("query %q: expected no hits, got %d", tc.query, len(hits))
		}
	}
}

func TestBM25SearchKindFilter(t *testing.T) {
	entries := []Entry{
		sampleNameEntry(),
		{
			ID:         "shell-exit-42",
			Kind:       KindError,
			Title:      "Shell exit 42",
			RecallText: "shell command exits with status 42",
		},
	}
	hits := searchBM25(entries, "shell exit", string(KindError), defaultSynonymGroups)
	if len(hits) != 1 || hits[0].ID != "shell-exit-42" {
		t.Fatalf("unexpected hits: %+v", hits)
	}
	hits = searchBM25(entries, "shell exit", string(KindPreference), defaultSynonymGroups)
	if len(hits) != 0 {
		t.Fatalf("expected kind filter to exclude error entry, got %+v", hits)
	}
}

func TestBM25SearchRanking(t *testing.T) {
	entries := []Entry{
		{
			ID:            "generic-name",
			Kind:          KindPreference,
			Title:         "Naming convention",
			RecallText:    "Use descriptive names for Leonard's modules.",
			RecallQueries: []string{"user name Leonard"},
		},
		sampleNameEntry(),
	}
	hits := searchBM25(entries, "user name Leonard", string(KindPreference), defaultSynonymGroups)
	if len(hits) < 2 {
		t.Fatalf("expected at least 2 hits, got %d", len(hits))
	}
	if hits[0].ID != "name-des-benutzers" {
		t.Fatalf("expected name entry first, got %q", hits[0].ID)
	}
	if hits[0].Score <= hits[1].Score {
		t.Fatalf("expected descending scores, got %.3f <= %.3f", hits[0].Score, hits[1].Score)
	}
}

func TestTermMatchesPrefixStem(t *testing.T) {
	if !termMatches("heiße", "heißt") {
		t.Fatal("expected heiße to match heißt via prefix")
	}
	if termMatches("abc", "xyz") {
		t.Fatal("expected unrelated terms not to match")
	}
}

func TestTokenizeSkipsStopWords(t *testing.T) {
	toks := tokenize("Wie heiße ich?")
	if len(toks) != 1 || toks[0] != "heiße" {
		t.Fatalf("unexpected tokens: %v", toks)
	}
}

func TestEntryDocumentIncludesTags(t *testing.T) {
	doc := entryDocument(Entry{
		Title:         "T",
		RecallText:    "R",
		Tags:          []string{"python", "bench"},
		Keywords:      []string{"keyword"},
		RecallQueries: []string{"how to run"},
		VerifiedFix:   "fix",
	})
	for _, want := range []string{"T", "R", "python", "bench", "keyword", "how to run", "fix"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("document missing %q: %q", want, doc)
		}
	}
}

func TestBM25SearchUsesRecallQueries(t *testing.T) {
	entries := []Entry{{
		ID:            "name-entry",
		Kind:          KindPreference,
		Title:         "User identity",
		RecallText:    "Stored user fact.",
		RecallQueries: []string{"wie ist mein name"},
	}}
	hits := searchBM25(entries, "wie ist mein name", string(KindPreference), defaultSynonymGroups)
	if len(hits) == 0 {
		t.Fatal("expected recall_queries to match")
	}
}
