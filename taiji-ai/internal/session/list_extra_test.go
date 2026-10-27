package session

import (
	"testing"
	"time"
)

func TestListAllMissingSessionsDir(t *testing.T) {
	root := t.TempDir()
	summaries, err := ListAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if summaries != nil {
		t.Fatalf("expected nil slice, got %#v", summaries)
	}
}

func TestListAllSortsNewestFirst(t *testing.T) {
	root := t.TempDir()
	old, err := New(root, "old", "p", "chat")
	if err != nil {
		t.Fatal(err)
	}
	old.State.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	if err := old.Save(); err != nil {
		t.Fatal(err)
	}

	newer, err := New(root, "new", "p", "chat")
	if err != nil {
		t.Fatal(err)
	}
	newer.State.CreatedAt = time.Now().UTC()
	if err := newer.Save(); err != nil {
		t.Fatal(err)
	}

	summaries, err := ListAll(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 {
		t.Fatalf("count = %d", len(summaries))
	}
	if summaries[0].ID != "new" || summaries[1].ID != "old" {
		t.Fatalf("order = %+v", summaries)
	}
}
