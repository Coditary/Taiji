package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteRemovesSessionAndBinding(t *testing.T) {
	root := t.TempDir()
	workDir := filepath.Join(root, "src")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatal(err)
	}

	sess, err := New(root, "to-delete", "", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveDirBinding(root, workDir, sess.State.ID); err != nil {
		t.Fatal(err)
	}

	if err := Delete(root, sess.State.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(sess.Dir); !os.IsNotExist(err) {
		t.Fatalf("session dir still exists: %v", err)
	}
	if _, ok, err := LoadDirBinding(root, workDir); err != nil || ok {
		t.Fatalf("binding still present: ok=%v err=%v", ok, err)
	}
}

func TestDeleteUnknownSession(t *testing.T) {
	root := t.TempDir()
	if err := Delete(root, "missing"); err == nil {
		t.Fatal("expected error for missing session")
	}
}
