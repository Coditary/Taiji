package agentsdoc

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadMissing(t *testing.T) {
	dir := t.TempDir()
	content, found, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if found || content != "" {
		t.Fatalf("expected missing file, got found=%v content=%q", found, content)
	}
}

func TestLoadPresent(t *testing.T) {
	dir := t.TempDir()
	want := "Use Go conventions.\nRun tests before committing."
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(want), 0o644); err != nil {
		t.Fatal(err)
	}
	content, found, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !found || content != want {
		t.Fatalf("got found=%v content=%q", found, content)
	}
	appendix := SystemAppendix(content)
	if !strings.Contains(appendix, want) {
		t.Fatalf("appendix missing content: %q", appendix)
	}
}
