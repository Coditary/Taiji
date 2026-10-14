package agent

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/coditary/taiji/internal/provider"
)

const embeddedToolCallMarker = "[tool call]"

type embeddedToolSpan struct {
	start int
	end   int
	call  provider.ToolCall
}

// ParseEmbeddedToolCalls extracts tool invocations written as plain text in a trailing
// block, e.g. "[tool call] read_file {"path": "Makefile"}".
func ParseEmbeddedToolCalls(content string) []provider.ToolCall {
	spans := findEmbeddedToolSpans(content)
	if len(spans) == 0 {
		return nil
	}
	out := make([]provider.ToolCall, len(spans))
	for i, span := range spans {
		out[i] = span.call
	}
	return out
}

// StripEmbeddedToolCalls removes a trailing embedded tool-call block and any
// premature answer text after it from assistant text.
func StripEmbeddedToolCalls(content string) string {
	spans := findEmbeddedToolSpans(content)
	if len(spans) == 0 {
		return strings.TrimSpace(content)
	}
	return strings.TrimSpace(content[:spans[0].start])
}

// StreamPreambleBeforeToolCalls returns assistant prose shown before tool calls.
// It prefers parsed preamble but falls back to raw streamed text before the marker.
func StreamPreambleBeforeToolCalls(streamed, respContent string) string {
	raw := streamed
	if strings.TrimSpace(raw) == "" {
		raw = respContent
	}
	if preamble := StripEmbeddedToolCalls(raw); preamble != "" {
		return preamble
	}
	lower := strings.ToLower(raw)
	idx := strings.Index(lower, embeddedToolCallMarker)
	if idx > 0 {
		return strings.TrimSpace(raw[:idx])
	}
	return ""
}

func findEmbeddedToolSpans(content string) []embeddedToolSpan {
	all := findAllEmbeddedToolSpans(content)
	if len(all) == 0 {
		return nil
	}
	return trailingEmbeddedToolSpans(content, all)
}

func trailingEmbeddedToolSpans(content string, spans []embeddedToolSpan) []embeddedToolSpan {
	end := len(spans) - 1
	start := end
	for i := end; i > 0; i-- {
		between := content[spans[i-1].end : spans[i].start]
		if strings.TrimSpace(between) != "" {
			break
		}
		start = i - 1
	}
	return spans[start : end+1]
}

func findAllEmbeddedToolSpans(content string) []embeddedToolSpan {
	if content == "" {
		return nil
	}
	lower := strings.ToLower(content)
	var spans []embeddedToolSpan
	search := 0
	id := 0
	for {
		idx := strings.Index(lower[search:], embeddedToolCallMarker)
		if idx < 0 {
			break
		}
		start := search + idx
		afterMarker := start + len(embeddedToolCallMarker)
		rest := strings.TrimLeft(content[afterMarker:], " \t")
		leading := afterMarker + (len(content[afterMarker:]) - len(rest))
		name, args, ok := parseEmbeddedToolNameAndArgs(rest)
		if !ok {
			search = start + len(embeddedToolCallMarker)
			continue
		}
		afterName := strings.TrimLeft(rest[len(name):], " \t")
		end := leading + (len(rest) - len(afterName)) + len(args)
		id++
		spans = append(spans, embeddedToolSpan{
			start: start,
			end:   end,
			call: provider.ToolCall{
				ID:   fmt.Sprintf("call_text_%d", id),
				Name: name,
				Args: args,
			},
		})
		search = end
	}
	return spans
}

func parseEmbeddedToolNameAndArgs(rest string) (name, args string, ok bool) {
	rest = strings.TrimSpace(rest)
	if rest == "" {
		return "", "", false
	}
	i := 0
	for i < len(rest) && !unicode.IsSpace(rune(rest[i])) && rest[i] != '{' {
		i++
	}
	name = strings.TrimSpace(rest[:i])
	if name == "" {
		return "", "", false
	}
	rest = strings.TrimSpace(rest[i:])
	if !strings.HasPrefix(rest, "{") {
		return "", "", false
	}
	args, ok = extractBalancedJSON(rest, '{', '}')
	if !ok {
		return "", "", false
	}
	return name, args, true
}

func extractBalancedJSON(s string, open, close byte) (string, bool) {
	if len(s) == 0 || s[0] != open {
		return "", false
	}
	depth := 0
	inString := false
	escaped := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if c == '\\' {
				escaped = true
				continue
			}
			if c == '"' {
				inString = false
			}
			continue
		}
		switch c {
		case '"':
			inString = true
		case open:
			depth++
		case close:
			depth--
			if depth == 0 {
				return s[:i+1], true
			}
		}
	}
	return "", false
}
