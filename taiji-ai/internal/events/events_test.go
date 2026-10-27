package events

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriterPathNil(t *testing.T) {
	var w *Writer
	if w.Path() != "" {
		t.Fatal("nil writer path")
	}
}

func TestWriterEmitSubscriberOnly(t *testing.T) {
	var seen Type
	w := NewWriter("")
	w.Subscribe(func(ev Event) { seen = ev.Type })
	if err := w.Emit(Event{Type: TypeDone, Text: "ok"}); err != nil {
		t.Fatal(err)
	}
	if seen != TypeDone {
		t.Fatalf("seen=%q", seen)
	}
}

func TestWriterEmitPersistsJSONL(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "stream.jsonl")
	w := NewWriter(path)
	if err := w.Emit(Event{Type: TypeMessage, Text: "hello"}); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	body := string(raw)
	if !strings.Contains(body, `"type":"message"`) || !strings.Contains(body, "hello") {
		t.Fatalf("jsonl=%q", body)
	}
}

func TestWriterPathNonNil(t *testing.T) {
	w := NewWriter("/tmp/events.jsonl")
	if w.Path() != "/tmp/events.jsonl" {
		t.Fatalf("path=%q", w.Path())
	}
}

func TestWriterMultipleSubscribers(t *testing.T) {
	var n int
	w := NewWriter("")
	w.Subscribe(func(ev Event) { n++ })
	w.Subscribe(func(ev Event) { n += 10 })
	if err := w.Emit(Event{Type: TypeMessage}); err != nil {
		t.Fatal(err)
	}
	if n != 11 {
		t.Fatalf("n=%d", n)
	}
}

func TestWriterEmitSetsTimestamp(t *testing.T) {
	w := NewWriter("")
	var ts int64
	w.Subscribe(func(ev Event) {
		if !ev.Timestamp.IsZero() {
			ts = ev.Timestamp.Unix()
		}
	})
	if err := w.Emit(Event{Type: TypeStep}); err != nil {
		t.Fatal(err)
	}
	if ts == 0 {
		t.Fatal("expected timestamp")
	}
}
