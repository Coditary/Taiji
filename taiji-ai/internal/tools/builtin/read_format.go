package builtin

import (
	"fmt"
	"strings"
)

const (
	readFileLargeThreshold = 300
	readFileDefaultWindow  = 200
)

func formatFileContentWithLineNumbers(content string, startLine, endLine int) (string, int, bool) {
	lines := splitContentLines(content)
	total := len(lines)
	if total == 0 {
		return "", 0, false
	}

	start := 1
	end := total
	truncated := false

	if startLine > 0 {
		start = startLine
	}
	if endLine > 0 {
		end = endLine
	}
	if startLine <= 0 && endLine <= 0 && total > readFileLargeThreshold {
		end = readFileDefaultWindow
		truncated = end < total
	}
	if start < 1 {
		start = 1
	}
	if end > total {
		end = total
	}
	if start > end {
		return "", total, truncated
	}
	if start > 1 || end < total {
		truncated = true
	}

	var b strings.Builder
	for i := start; i <= end; i++ {
		fmt.Fprintf(&b, "%4d | %s\n", i, lines[i-1])
	}
	out := strings.TrimRight(b.String(), "\n")
	if truncated {
		out += fmt.Sprintf("\n\n[Showing lines %d-%d of %d. Use start_line/end_line to read more.]", start, end, total)
	}
	return out, total, truncated
}
