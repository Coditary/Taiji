package cli

import (
	"io"
	"os"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/events"
)

func capturePrintEvent(t *testing.T, ev events.Event) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	printEvent(w, ev)
	_ = w.Close()
	raw, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestPrintEventTypes(t *testing.T) {
	if out := capturePrintEvent(t, events.Event{Type: events.TypeStep, Text: "skip me"}); out != "" {
		t.Fatalf("step should be suppressed, got %q", out)
	}
	out := capturePrintEvent(t, events.Event{Type: events.TypeContinue, Text: "continuing"})
	out += capturePrintEvent(t, events.Event{Type: events.TypeThought, Step: 2, Text: "plan"})
	out += capturePrintEvent(t, events.Event{Type: events.TypeMessage, Text: "hello"})
	out += capturePrintEvent(t, events.Event{Type: events.TypeToolCall, ToolName: "grep", Args: `{"pattern":"x"}`})
	out += capturePrintEvent(t, events.Event{
		Type:     events.TypeObservation,
		ToolName: "edit_file",
		Text:     "ok",
		Data:     map[string]any{"diff": "+line"},
	})
	out += capturePrintEvent(t, events.Event{Type: events.TypeMemoryStored, Text: "saved fact"})
	out += capturePrintEvent(t, events.Event{Type: events.TypeTurnMetrics, Data: map[string]any{"tokens": 1}})
	out += capturePrintEvent(t, events.Event{Type: events.TypeError, Text: "boom"})

	for _, needle := range []string{
		"[continuing]",
		"thinking (step 2)",
		"assistant: hello",
		"tool grep",
		"observation edit_file",
		"+line",
		"memory_stored",
		"metrics:",
		"error: boom",
	} {
		if !strings.Contains(out, needle) {
			t.Fatalf("missing %q in:\n%s", needle, out)
		}
	}
	if strings.Contains(out, "skip me") {
		t.Fatalf("step events should be suppressed: %s", out)
	}
}
