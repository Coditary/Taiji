package agent

import (
	"strings"
	"testing"
)

func TestParseThinkingTags(t *testing.T) {
	raw := "prefix" + thinkingOpenTag + "Step one.\nStep two." + thinkingCloseTag + "Final answer."
	think, answer := parseThinkingTags(raw)
	if think != "Step one.\nStep two." {
		t.Fatalf("thinking = %q", think)
	}
	if answer != "prefix\n\nFinal answer." {
		t.Fatalf("answer = %q", answer)
	}

	_, answer = parseThinkingTags("No tags here.")
	if answer != "No tags here." {
		t.Fatalf("plain = %q", answer)
	}
}

func TestSanitizeChainStreamGluedContinue(t *testing.T) {
	raw := defaultContinueTrigger + thinkingOpenTag + "step two" + thinkingCloseTag
	sanitized := sanitizeChainStream(raw)
	if !strings.Contains(sanitized, thinkingOpenTag) {
		t.Fatalf("expected think tag after sanitize: %q", sanitized)
	}
	blocks := parseAllThinkingBlocks(sanitized)
	if len(blocks) != 1 || blocks[0] != "step two" {
		t.Fatalf("blocks = %v", blocks)
	}
}

func TestParseAllThinkingBlocksChained(t *testing.T) {
	raw := thinkingOpenTag + "First step." + thinkingCloseTag + "\n" + defaultContinueTrigger + "\n" +
		thinkingOpenTag + "Second step." + thinkingCloseTag + "\nFinal answer."
	blocks := parseAllThinkingBlocks(raw)
	if len(blocks) != 2 || blocks[0] != "First step." || blocks[1] != "Second step." {
		t.Fatalf("blocks = %v", blocks)
	}
	_, answer := parseThinkingTags(raw)
	if answer != "Final answer." {
		t.Fatalf("answer = %q", answer)
	}
}

func TestThinkingStreamParserChained(t *testing.T) {
	var thoughts []string
	var answers strings.Builder
	headers := 0
	parser := newThinkingStreamParser(0, false,
		func(int) error { headers++; return nil },
		func(s string) error { thoughts = append(thoughts, s); return nil },
		func(s string) error { answers.WriteString(s); return nil },
	)

	raw := thinkingOpenTag + "A." + thinkingCloseTag + "\n" + defaultContinueTrigger + "\n" +
		thinkingOpenTag + "B." + thinkingCloseTag + "\nDone."
	if err := parser.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := parser.Flush(); err != nil {
		t.Fatal(err)
	}
	if headers != 2 {
		t.Fatalf("headers = %d, want 2", headers)
	}
	if len(thoughts) != 2 || thoughts[0] != "A." || thoughts[1] != "B." {
		t.Fatalf("thoughts = %v", thoughts)
	}
	if answers.String() != "Done." {
		t.Fatalf("answer = %q", answers.String())
	}
}

func TestThinkingStreamParserGluedContinue(t *testing.T) {
	var thoughts []string
	var answers strings.Builder
	parser := newThinkingStreamParser(0, false,
		func(int) error { return nil },
		func(s string) error { thoughts = append(thoughts, s); return nil },
		func(s string) error { answers.WriteString(s); return nil },
	)

	raw := thinkingOpenTag + "First." + thinkingCloseTag + "<|continue|" + thinkingOpenTag + "Second." + thinkingCloseTag + " Answer."
	if err := parser.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := parser.Flush(); err != nil {
		t.Fatal(err)
	}
	if len(thoughts) != 2 {
		t.Fatalf("thoughts = %v", thoughts)
	}
	if answers.String() != "Answer." {
		t.Fatalf("answer = %q", answers.String())
	}
}

func TestThinkingStreamParser(t *testing.T) {
	var thinking, answer strings.Builder
	headers := 0
	parser := newThinkingStreamParser(0, false,
		func(int) error { headers++; return nil },
		func(s string) error { thinking.WriteString(s); return nil },
		func(s string) error { answer.WriteString(s); return nil },
	)

	raw := "Hello! " + thinkingOpenTag + "First thought." + thinkingCloseTag + "Hi there!"
	if err := parser.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := parser.Flush(); err != nil {
		t.Fatal(err)
	}
	if headers != 1 {
		t.Fatalf("headers = %d, want 1", headers)
	}
	if thinking.String() != "First thought." {
		t.Fatalf("thinking = %q", thinking.String())
	}
	if answer.String() != "Hello! Hi there!" {
		t.Fatalf("answer = %q", answer.String())
	}
}

func TestParseThinkingTagsRedactedAlias(t *testing.T) {
	open := "<" + "redacted_thinking" + ">"
	close := "</" + "redacted_thinking" + ">"
	raw := open + "internal thought." + close + "Final."
	think, answer := parseThinkingTags(raw)
	if think != "internal thought." {
		t.Fatalf("thinking = %q", think)
	}
	if answer != "Final." {
		t.Fatalf("answer = %q", answer)
	}
}

func TestParseThinkingTagsAnswerBeforeBlock(t *testing.T) {
	raw := "Visible answer." + thinkingOpenTag + "duplicate" + thinkingCloseTag
	think, answer := parseThinkingTags(raw)
	if think != "duplicate" {
		t.Fatalf("thinking = %q", think)
	}
	if answer != "Visible answer." {
		t.Fatalf("answer = %q", answer)
	}
}

func TestThinkingStreamParserOnePerCallDiscardsExtraThoughts(t *testing.T) {
	var thoughts []string
	var answers strings.Builder
	headers := 0
	parser := newThinkingStreamParser(0, true,
		func(int) error { headers++; return nil },
		func(s string) error { thoughts = append(thoughts, s); return nil },
		func(s string) error { answers.WriteString(s); return nil },
	)

	raw := thinkingOpenTag + "First." + thinkingCloseTag + "\n" + defaultContinueTrigger + "\n" +
		thinkingOpenTag + "Second." + thinkingCloseTag + "\nGarbage answer."
	if err := parser.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := parser.Flush(); err != nil {
		t.Fatal(err)
	}
	if headers != 1 || len(thoughts) != 1 || thoughts[0] != "First." {
		t.Fatalf("headers=%d thoughts=%v", headers, thoughts)
	}
	if answers.String() != "" {
		t.Fatalf("answer should be empty, got %q", answers.String())
	}
}

func TestThinkingStreamParserOnePerCallAllowsFinalAnswer(t *testing.T) {
	var answers strings.Builder
	parser := newThinkingStreamParser(0, true,
		func(int) error { return nil },
		func(string) error { return nil },
		func(s string) error { answers.WriteString(s); return nil },
	)

	raw := thinkingOpenTag + "Done thinking." + thinkingCloseTag + "\nFinal answer here."
	if err := parser.Write(raw); err != nil {
		t.Fatal(err)
	}
	if err := parser.Flush(); err != nil {
		t.Fatal(err)
	}
	if answers.String() != "Final answer here." {
		t.Fatalf("answer = %q", answers.String())
	}
}

func TestPartialTagSuffix(t *testing.T) {
	partial := thinkingOpenTag[:len(thinkingOpenTag)-2]
	if partialTagSuffix(partial, thinkingOpenTag) != len(partial) {
		t.Fatalf("expected partial open tag holdback for %q", partial)
	}
	if partialTagSuffix("hello", thinkingOpenTag) != 0 {
		t.Fatal("expected no holdback")
	}
}
