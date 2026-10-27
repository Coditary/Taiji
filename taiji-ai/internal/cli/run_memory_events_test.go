package cli

import (
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/events"
)

func TestPrintEventMemoryAndEmptyMessage(t *testing.T) {
	out := capturePrintEvent(t, events.Event{Type: events.TypeMemoryRecalled, Text: "fact"})
	out += capturePrintEvent(t, events.Event{Type: events.TypeMemoryRevised, Text: "updated"})
	out += capturePrintEvent(t, events.Event{Type: events.TypeMessage, Text: "   "})
	if !strings.Contains(out, "memory_recalled") || !strings.Contains(out, "memory_revised") {
		t.Fatalf("out=%q", out)
	}
	if strings.Contains(out, "assistant:") {
		t.Fatal("blank message should not print assistant line")
	}
}
