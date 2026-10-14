package memory

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestJSONStoreSaveLoad(t *testing.T) {
	dir := t.TempDir()
	store := NewStore("", dir)

	saved, err := store.Save("local", Entry{
		Kind:        KindPersona,
		Title:       "Assistant name: Luma",
		RecallText:  "The assistant's name is Luma.",
		VerifiedFix: "Luma",
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.ID == "" {
		t.Fatal("expected id")
	}

	indexPath := filepath.Join(dir, entriesFileName)
	raw, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"kind":"persona"`) && !strings.Contains(string(raw), `"kind": "persona"`) {
		t.Fatalf("expected compact json index, got %q", string(raw))
	}

	found, err := store.FindByID(saved.ID)
	if err != nil {
		t.Fatal(err)
	}
	if found.VerifiedFix != "Luma" {
		t.Fatalf("verified fix=%q", found.VerifiedFix)
	}

	ragPath := filepath.Join(dir, ragIndexSubdir, saved.ID+".json")
	if _, err := os.Stat(ragPath); err != nil {
		t.Fatalf("rag sidecar missing: %v", err)
	}
}

func TestImportLegacyMarkdownIntoJSON(t *testing.T) {
	dir := t.TempDir()
	md := filepath.Join(dir, "user-name.md")
	content := "---\nkind: preference\ntitle: User name\ntags: [name]\nrecall_text: User is Leonard\n---\n\n# Verified fix\n\nLeonard\n"
	if err := os.WriteFile(md, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	store := NewStore("", dir)
	all, err := store.ListAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("entries=%d", len(all))
	}
	if all[0].VerifiedFix != "Leonard" {
		t.Fatalf("fix=%q", all[0].VerifiedFix)
	}
	if _, err := os.Stat(md); !os.IsNotExist(err) {
		t.Fatal("expected legacy md removed after import")
	}
}
