package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestStoreSaveLoadAndRevise(t *testing.T) {
	dir := t.TempDir()
	global := filepath.Join(dir, "global")
	project := filepath.Join(dir, "project")
	store := NewStore(global, project)

	saved, err := store.Save("local", Entry{
		Kind:        KindError,
		Title:       "Go undefined identifier",
		Fingerprint: "go:undefined:*",
		RecallText:  "undefined symbol during go build",
		VerifiedFix: "1. grep symbol\n2. fix import",
		LastVerified: "2026-09-24",
		RevisionLog: "- 2026-09-24: created",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(saved.Path, entriesFileName+"#") {
		t.Fatalf("unexpected path %q", saved.Path)
	}

	loaded, err := LoadFile(saved.Path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Kind != KindError {
		t.Fatalf("kind=%q", loaded.Kind)
	}
	if loaded.VerifiedFix == "" {
		t.Fatal("expected verified fix section")
	}

	found, err := store.FindByID(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	found.Hits = 2
	found.RevisionLog = appendRevisionLog(found.RevisionLog, "Updated fix")
	found.VerifiedFix = "1. grep\n2. rename"
	updated, err := store.Save(found.Scope, found)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Hits != 2 {
		t.Fatalf("hits=%d", updated.Hits)
	}
	loaded, err = LoadFile(updated.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(loaded.RevisionLog, "Updated fix") {
		t.Fatalf("missing revision note in %q", loaded.RevisionLog)
	}
	indexRaw, err := os.ReadFile(filepath.Join(project, entriesFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(indexRaw), "Updated fix") {
		t.Fatalf("missing revision note in index %q", string(indexRaw))
	}
}

func TestRecallExactFingerprint(t *testing.T) {
	dir := t.TempDir()
	project := taijicfg.ProjectMemoryDir(dir)
	store := NewStore("", project)
	_, err := store.Save("local", Entry{
		Kind:        KindError,
		Title:       "ENOENT package",
		Fingerprint: "npm:enoent:*",
		RecallText:  "npm install missing module",
		VerifiedFix: "npm install",
	})
	if err != nil {
		t.Fatal(err)
	}

	cfg := &taijicfg.Config{
		Root: dir,
		Memory: taijicfg.MemoryConfig{
			GlobalDir: filepath.Join(dir, "global"),
		},
	}
	svc := NewService(cfg, nil)
	entries, err := svc.Recall(t.Context(), RecallRequest{
		Query:       "npm ERR! enoent some-package",
		Fingerprint: "npm err enoent some-package",
		Kind:        string(KindError),
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].MatchSource != "exact" {
		t.Fatalf("match source=%q", entries[0].MatchSource)
	}
}
