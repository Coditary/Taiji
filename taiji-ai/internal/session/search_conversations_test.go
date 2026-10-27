package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func TestSearchConversationsEmptyQuery(t *testing.T) {
	hits, err := SearchConversations(t.TempDir(), "  ", 5)
	if err != nil || hits != nil {
		t.Fatalf("hits=%v err=%v", hits, err)
	}
}

func TestSearchConversationsMultiKeyword(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "findme", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	s.State.Messages = append(s.State.Messages,
		provider.Message{Role: provider.RoleUser, Content: "The blue dragon flew east"},
		provider.Message{Role: provider.RoleAssistant, Content: "Noted."},
	)
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	hits, err := SearchConversations(root, "blue dragon", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || !strings.Contains(hits[0].Snippet, "blue") {
		t.Fatalf("hits=%#v", hits)
	}
	hits, err = SearchConversations(root, "blue moon", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected no match, got %#v", hits)
	}
}

func TestSearchConversations(t *testing.T) {
	root := t.TempDir()
	sessDir := filepath.Join(config.SessionsDir(root), "abc")
	if err := os.MkdirAll(sessDir, 0o755); err != nil {
		t.Fatal(err)
	}
	state := State{
		ID:        "abc",
		Name:      "Demo",
		CreatedAt: time.Now().UTC(),
		Messages: []provider.Message{
			{Role: provider.RoleUser, Content: "How do we deploy Taiji?"},
			{Role: provider.RoleAssistant, Content: "Use make install."},
		},
	}
	raw, _ := json.Marshal(state)
	if err := os.WriteFile(filepath.Join(sessDir, "state.json"), raw, 0o644); err != nil {
		t.Fatal(err)
	}
	hits, err := SearchConversations(root, "deploy taiji", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 1 || hits[0].SessionID != "abc" {
		t.Fatalf("hits=%v", hits)
	}
}
