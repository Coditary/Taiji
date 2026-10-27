package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestImportCharaCardJSON(t *testing.T) {
	root := t.TempDir()
	templates := filepath.Join(root, ".taiji", "templates")
	cardPath := filepath.Join(root, "card.json")
	raw := `{
  "data": {
    "name": "Test Hero",
    "description": "A brave tester.",
    "personality": "calm",
    "scenario": "lab",
    "first_mes": "Hello!",
    "mes_example": "<START>\n{{user}}: Hi\n{{char}}: Hey"
  }
}`
	if err := os.WriteFile(cardPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	tplPath, lore, err := ImportCharaCard(root, cardPath, templates)
	if err != nil {
		t.Fatal(err)
	}
	if lore != "" {
		t.Fatalf("unexpected lore pack %q", lore)
	}
	if !strings.HasSuffix(tplPath, "test-hero.md") {
		t.Fatalf("tplPath=%q", tplPath)
	}
	body, err := os.ReadFile(tplPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, "Test Hero") || !strings.Contains(text, "<example>") {
		t.Fatalf("body=%q", text)
	}
}

func TestImportCharaCardWithLoreBook(t *testing.T) {
	root := t.TempDir()
	templates := filepath.Join(root, ".taiji", "templates")
	cardPath := filepath.Join(root, "card.json")
	raw := `{
  "data": {
    "name": "Dragon Keeper",
    "description": "Knows dragons.",
    "character_book": {
      "entries": [
        {"name": "dragons", "keys": ["dragon"], "content": "Ancient wyrms.", "enabled": true}
      ]
    }
  }
}`
	if err := os.WriteFile(cardPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	_, lore, err := ImportCharaCard(root, cardPath, templates)
	if err != nil {
		t.Fatal(err)
	}
	if lore != "dragon-keeper-lore" {
		t.Fatalf("lore=%q", lore)
	}
	packPath := filepath.Join(root, ".taiji", "packs", lore, "dragons.md")
	if _, err := os.Stat(packPath); err != nil {
		t.Fatalf("pack file: %v", err)
	}
}

func TestImportCharaCardFlatJSON(t *testing.T) {
	root := t.TempDir()
	templates := filepath.Join(root, ".taiji", "templates")
	cardPath := filepath.Join(root, "flat.json")
	raw := `{"name": "Flat Hero", "description": "No wrapper."}`
	if err := os.WriteFile(cardPath, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	tplPath, _, err := ImportCharaCard(root, cardPath, templates)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(tplPath, "flat-hero.md") {
		t.Fatalf("tpl=%q", tplPath)
	}
}
