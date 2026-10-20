package builtin

import (
	"fmt"
	"strings"
)

const editFileDiffMaxLines = 30

type fileEdit struct {
	OldText string
	NewText string
}

// applyFileEdits applies edits sequentially to content. Each old_text must match exactly once
// (with optional per-line whitespace tolerance).
func applyFileEdits(content string, edits []fileEdit) (string, []int, error) {
	if len(edits) == 0 {
		return "", nil, fmt.Errorf("edits must contain at least one entry")
	}
	current := content
	var changedLines []int
	for i, ed := range edits {
		if strings.TrimSpace(ed.OldText) == "" {
			return "", nil, fmt.Errorf("edit %d: old_text is required", i+1)
		}
		replaced, line, err := replaceOnce(current, ed.OldText, ed.NewText)
		if err != nil {
			return "", nil, fmt.Errorf("edit %d: %w", i+1, err)
		}
		current = replaced
		if line > 0 {
			changedLines = append(changedLines, line)
		}
	}
	return current, changedLines, nil
}

func replaceOnce(content, oldText, newText string) (string, int, error) {
	count := strings.Count(content, oldText)
	if count == 1 {
		idx := strings.Index(content, oldText)
		line := lineNumberAt(content, idx)
		return strings.Replace(content, oldText, newText, 1), line, nil
	}
	if count > 1 {
		return "", 0, fmt.Errorf("old_text matched %d times; include more surrounding context so it is unique\n%s",
			count, lineContextForNeedle(content, oldText))
	}
	replaced, line, ok := replaceOnceWhitespaceTolerant(content, oldText, newText)
	if ok {
		return replaced, line, nil
	}
	return "", 0, fmt.Errorf("old_text not found\n%s", lineContextForNeedle(content, oldText))
}

func replaceOnceWhitespaceTolerant(content, oldText, newText string) (string, int, bool) {
	oldLines := splitContentLines(oldText)
	if len(oldLines) == 0 {
		return "", 0, false
	}
	fileLines := splitContentLines(content)
	if len(fileLines) == 0 {
		return "", 0, false
	}
	var matches []int
	for start := 0; start <= len(fileLines)-len(oldLines); start++ {
		if linesMatchTrimmed(fileLines[start:start+len(oldLines)], oldLines) {
			matches = append(matches, start)
		}
	}
	if len(matches) != 1 {
		return "", 0, false
	}
	start := matches[0]
	end := start + len(oldLines)
	newLines := splitContentLines(newText)
	merged := append(append(fileLines[:start], newLines...), fileLines[end:]...)
	lineNum := start + 1
	return joinContentLines(merged), lineNum, true
}

func linesMatchTrimmed(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

func splitContentLines(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func joinContentLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n")
}

func lineNumberAt(content string, idx int) int {
	if idx < 0 {
		return 0
	}
	return strings.Count(content[:idx], "\n") + 1
}

func lineContextForNeedle(content, needle string) string {
	needle = strings.TrimSpace(needle)
	if needle == "" {
		return "(empty old_text)"
	}
	first := strings.Split(strings.ReplaceAll(needle, "\r\n", "\n"), "\n")[0]
	first = strings.TrimSpace(first)
	if first == "" {
		return "(no context)"
	}
	fileLines := splitContentLines(content)
	var out []string
	for i, line := range fileLines {
		if strings.Contains(line, first) || strings.Contains(strings.TrimSpace(line), first) {
			start := max(0, i-2)
			end := min(len(fileLines), i+3)
			for j := start; j < end; j++ {
				out = append(out, fmt.Sprintf("%4d | %s", j+1, fileLines[j]))
			}
			return strings.Join(out, "\n")
		}
	}
	if len(fileLines) > 8 {
		var head []string
		for i := 0; i < 4; i++ {
			head = append(head, fmt.Sprintf("%4d | %s", i+1, fileLines[i]))
		}
		head = append(head, "    | ...")
		for i := len(fileLines) - 3; i < len(fileLines); i++ {
			head = append(head, fmt.Sprintf("%4d | %s", i+1, fileLines[i]))
		}
		return strings.Join(head, "\n")
	}
	var all []string
	for i, line := range fileLines {
		all = append(all, fmt.Sprintf("%4d | %s", i+1, line))
	}
	return strings.Join(all, "\n")
}

func parseFileEdits(args map[string]any) ([]fileEdit, error) {
	raw, ok := args["edits"]
	if !ok {
		return nil, fmt.Errorf("edits is required")
	}
	list, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("edits must be an array")
	}
	out := make([]fileEdit, 0, len(list))
	for i, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("edit %d: must be an object", i+1)
		}
		oldText, _ := m["old_text"].(string)
		newText, _ := m["new_text"].(string)
		if strings.TrimSpace(oldText) == "" {
			return nil, fmt.Errorf("edit %d: old_text is required", i+1)
		}
		out = append(out, fileEdit{OldText: oldText, NewText: newText})
	}
	return out, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
