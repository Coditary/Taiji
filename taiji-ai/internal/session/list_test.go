package session

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestListAllReturnsSummaries(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "sess-1", "openai", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := sess.SetName("Named chat", true); err != nil {
		t.Fatal(err)
	}

	summaries, err := ListAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 {
		t.Fatalf("summaries = %d, want 1", len(summaries))
	}
	if summaries[0].ID != "sess-1" || summaries[0].Name != "Named chat" {
		t.Fatalf("summary = %+v", summaries[0])
	}
}

func TestListAllSkipsInvalidSessions(t *testing.T) {
	root := t.TempDir()
	sessionsDir := filepath.Join(root, ".taiji", "sessions")
	if err := os.MkdirAll(filepath.Join(sessionsDir, "broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionsDir, "broken", "state.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}

	valid, err := New(root, "good", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	valid.State.CreatedAt = time.Now().UTC()
	if err := valid.Save(); err != nil {
		t.Fatal(err)
	}

	summaries, err := ListAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].ID != "good" {
		t.Fatalf("summaries = %+v", summaries)
	}
}
