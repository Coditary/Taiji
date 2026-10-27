package events

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadWriteFileRoundTrip(t *testing.T) {
	if out, err := ReadFile(""); err != nil || out != nil {
		t.Fatalf("empty path: out=%v err=%v", out, err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "ev.jsonl")
	ts := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	evts := []Event{
		{Type: TypeMessage, Text: "hello", Timestamp: ts},
		{Type: TypeDone, Text: "ok"},
	}
	if err := WriteFile(path, evts); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil || len(got) != 2 {
		t.Fatalf("read: len=%d err=%v", len(got), err)
	}
	if got[0].Text != "hello" || got[1].Type != TypeDone {
		t.Fatalf("got=%#v", got)
	}
	if err := WriteFile("", evts); err == nil {
		t.Fatal("empty write path")
	}
}

func TestReadFileSkipsBadLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.jsonl")
	raw := `{"type":"message","text":"ok"}
not-json
{"type":"done"}
`
	if err := os.WriteFile(path, []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := ReadFile(path)
	if err != nil || len(got) != 2 {
		t.Fatalf("got=%d err=%v", len(got), err)
	}
}
