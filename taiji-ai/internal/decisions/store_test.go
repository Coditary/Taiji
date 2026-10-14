package decisions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/decision"
)

func TestSaveProfileWritesDECISIONmd(t *testing.T) {
	dir := t.TempDir()
	path, err := SaveProfile(dir, ProfileFromQuestions(
		"support-route",
		"Route support tickets",
		"User asks which team should handle an issue",
		[]decision.Question{
			{ID: "route", Type: decision.QuestionChoice, Instructions: "Which team?", Criteria: []string{"billing", "technical"}},
			{ID: "urgent", Type: decision.QuestionNoul, Instructions: "Is it urgent?"},
		},
		"",
	))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)
	if !strings.Contains(content, "name: support-route") {
		t.Fatalf("missing name: %q", content)
	}
	if !strings.Contains(content, "route:") || !strings.Contains(content, "urgent:") {
		t.Fatalf("missing questions: %q", content)
	}

	loaded, err := LoadFile(filepath.Join(dir, "support-route", decisionFileName))
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "support-route" || len(loaded.Questions) != 2 {
		t.Fatalf("loaded = %+v", loaded)
	}
}

func TestNormalizeProfileName(t *testing.T) {
	got, err := NormalizeProfileName("Support_Route")
	if err != nil || got != "support-route" {
		t.Fatalf("got %q err=%v", got, err)
	}
	if _, err := NormalizeProfileName(""); err == nil {
		t.Fatal("expected error for empty name")
	}
}
