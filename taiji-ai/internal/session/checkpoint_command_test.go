package session

import (
	"strings"
	"testing"
)

func TestParseCheckpointCommandExport(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "cp1", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseCheckpointCommand(s, root, "/checkpoint my-save")
	if err != nil || !handled || !strings.Contains(reply, "Checkpoint gespeichert") {
		t.Fatalf("checkpoint: reply=%q err=%v", reply, err)
	}
}

func TestParseCheckpointCommandExportSessionDefaultPath(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "ex1", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseCheckpointCommand(s, root, "/export-session")
	if err != nil || !handled || !strings.Contains(reply, "Session exportiert") {
		t.Fatalf("export: reply=%q err=%v", reply, err)
	}
}

func TestParseCheckpointCommandImportMissingFile(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "im1", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	_, handled, err := ParseCheckpointCommand(s, root, "/import-session /no/such/archive.taiji.zip")
	if err == nil || !handled {
		t.Fatalf("import: handled=%v err=%v", handled, err)
	}
}
