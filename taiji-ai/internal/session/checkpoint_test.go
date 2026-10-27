package session

import (
	"archive/zip"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestExportImportCheckpointRoundTrip(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	sess.State.Name = "Checkpoint test"
	sess.State.Messages = append(sess.State.Messages, provider.Message{Role: provider.RoleUser, Content: "hello"})

	dest := filepath.Join(root, "snap")
	path, err := ExportCheckpoint(sess, dest, "label")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, ".taiji.zip") {
		t.Fatalf("path=%q", path)
	}

	imported, err := ImportCheckpoint(root, path)
	if err != nil {
		t.Fatal(err)
	}
	if imported.State.Name != "Checkpoint test" || len(imported.State.Messages) != 1 {
		t.Fatalf("state=%+v", imported.State)
	}
}

func TestExportCheckpointManifest(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	path, err := ExportCheckpoint(sess, filepath.Join(root, "x.zip"), "lbl")
	if err != nil {
		t.Fatal(err)
	}
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	var manifest checkpointManifest
	for _, f := range r.File {
		if f.Name != "manifest.json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}
		if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
			t.Fatal(err)
		}
		_ = rc.Close()
	}
	if manifest.SessionID != sess.State.ID || manifest.Label != "lbl" {
		t.Fatalf("manifest=%+v", manifest)
	}
}

func TestIsDraftWithMessages(t *testing.T) {
	root := t.TempDir()
	sess, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if !IsDraft(sess) {
		t.Fatal("new session should be draft")
	}
	sess.State.Messages = append(sess.State.Messages, provider.Message{Role: provider.RoleUser, Content: "hi"})
	if IsDraft(sess) {
		t.Fatal("session with user message is not draft")
	}
}
