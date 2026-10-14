package agent

import (
	"strings"
	"testing"
)

func TestSafeEmbeddedToolStreamLen_ignoresJSONArrayBracket(t *testing.T) {
	content := `Preamble {"names": ["matrix-expansion-engine"]}`
	if got := safeEmbeddedToolStreamLen(content); got != len(content) {
		t.Fatalf("expected full preamble to stream, got %d of %d", got, len(content))
	}
}

func TestEmbeddedToolStreamFilterHoldsSuffix(t *testing.T) {
	var out strings.Builder
	filter := newEmbeddedToolStreamFilter(func(delta string) error {
		out.WriteString(delta)
		return nil
	})
	if err := filter.Write("Ich lade den Skill.\n\n[tool call] invoke_skill"); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Ich lade den Skill.\n\n" {
		t.Fatalf("unexpected streamed prefix: %q", out.String())
	}
}

func TestEmbeddedToolStreamFilterDiscardHeldSuffix(t *testing.T) {
	var out strings.Builder
	filter := newEmbeddedToolStreamFilter(func(delta string) error {
		out.WriteString(delta)
		return nil
	})
	_ = filter.Write("Hello [tool call] invoke_skill")
	filter.DiscardHeldSuffix()
	_ = filter.FlushRemaining()
	if out.String() != "Hello " {
		t.Fatalf("unexpected output: %q", out.String())
	}
}

func TestEmbeddedToolStreamFilterFlushRemaining(t *testing.T) {
	var out strings.Builder
	filter := newEmbeddedToolStreamFilter(func(delta string) error {
		out.WriteString(delta)
		return nil
	})
	_ = filter.Write("Hello world")
	if err := filter.FlushRemaining(); err != nil {
		t.Fatal(err)
	}
	if out.String() != "Hello world" {
		t.Fatalf("unexpected output: %q", out.String())
	}
}
