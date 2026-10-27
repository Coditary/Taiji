package diff

import (
	"fmt"
	"strings"
)

// Unified renders a small unified diff between old and new text.
// When maxLines > 0, output is truncated with a summary line.
func Unified(path string, oldText, newText string, maxLines int) string {
	oldLines := splitLines(oldText)
	newLines := splitLines(newText)
	if len(oldLines) == 0 && len(newLines) == 0 {
		return ""
	}

	var out []string
	out = append(out, "--- a/"+path)
	out = append(out, "+++ b/"+path)

	if len(oldLines) == 0 {
		out = append(out, "@@ -0,0 +1,"+itoa(len(newLines))+" @@")
		for _, line := range newLines {
			out = append(out, "+"+line)
		}
		return truncateLines(out, maxLines)
	}

	hunks := lineHunks(oldLines, newLines)
	if len(hunks) == 0 {
		return ""
	}
	for _, h := range hunks {
		out = append(out, fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.oldStart+1, h.oldLen, h.newStart+1, h.newLen))
		for _, line := range expandHunk(oldLines, newLines, h) {
			out = append(out, line)
		}
	}
	return truncateLines(out, maxLines)
}

type hunk struct {
	oldStart int
	oldLen   int
	newStart int
	newLen   int
	lines    []string
}

func windowSizeForChanges(changes int) int {
	if changes <= 0 {
		return 0
	}
	if changes%2 == 1 {
		return 5
	}
	return 6
}

func changeCount(h hunk) int {
	n := h.oldLen
	if h.newLen > n {
		n = h.newLen
	}
	return n
}

func expandHunk(oldLines, newLines []string, h hunk) []string {
	core := h.lines
	if len(core) == 0 {
		return core
	}
	win := windowSizeForChanges(changeCount(h))
	if win <= 0 || len(core) >= win {
		return core
	}
	before := (win - len(core)) / 2
	after := win - len(core) - before
	if before < 0 {
		before = 0
	}
	if after < 0 {
		after = 0
	}

	var out []string
	for k := before; k >= 1; k-- {
		oi := h.oldStart - k
		ni := h.newStart - k
		if oi >= 0 && ni >= 0 && oi < len(oldLines) && ni < len(newLines) && oldLines[oi] == newLines[ni] {
			out = append(out, " "+oldLines[oi])
		}
	}
	out = append(out, core...)
	for k := 0; k < after; k++ {
		oi := h.oldStart + h.oldLen + k
		ni := h.newStart + h.newLen + k
		if oi < len(oldLines) && ni < len(newLines) && oldLines[oi] == newLines[ni] {
			out = append(out, " "+oldLines[oi])
		} else {
			break
		}
	}
	return out
}

func lineHunks(oldLines, newLines []string) []hunk {
	lcs := longestCommonSubsequence(oldLines, newLines)
	var hunks []hunk
	i, j := 0, 0
	lcsIdx := 0

	for lcsIdx < len(lcs) {
		match := lcs[lcsIdx]
		if i < match.old || j < match.new {
			h := hunk{oldStart: i, newStart: j}
			for ; i < match.old; i++ {
				h.lines = append(h.lines, "-"+oldLines[i])
				h.oldLen++
			}
			for ; j < match.new; j++ {
				h.lines = append(h.lines, "+"+newLines[j])
				h.newLen++
			}
			if len(h.lines) > 0 {
				hunks = append(hunks, h)
			}
		}
		i = match.old + 1
		j = match.new + 1
		lcsIdx++
	}
	if i < len(oldLines) || j < len(newLines) {
		h := hunk{oldStart: i, newStart: j}
		for ; i < len(oldLines); i++ {
			h.lines = append(h.lines, "-"+oldLines[i])
			h.oldLen++
		}
		for ; j < len(newLines); j++ {
			h.lines = append(h.lines, "+"+newLines[j])
			h.newLen++
		}
		if len(h.lines) > 0 {
			hunks = append(hunks, h)
		}
	}
	return hunks
}

type lcsPoint struct {
	old int
	new int
}

func longestCommonSubsequence(a, b []string) []lcsPoint {
	n, m := len(a), len(b)
	dp := make([][]int, n+1)
	for i := range dp {
		dp[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if a[i] == b[j] {
				dp[i][j] = dp[i+1][j+1] + 1
			} else if dp[i+1][j] >= dp[i][j+1] {
				dp[i][j] = dp[i+1][j]
			} else {
				dp[i][j] = dp[i][j+1]
			}
		}
	}
	var out []lcsPoint
	i, j := 0, 0
	for i < n && j < m {
		if a[i] == b[j] {
			out = append(out, lcsPoint{old: i, new: j})
			i++
			j++
		} else if dp[i+1][j] >= dp[i][j+1] {
			i++
		} else {
			j++
		}
	}
	return out
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func truncateLines(lines []string, maxLines int) string {
	if maxLines <= 0 || len(lines) <= maxLines {
		return strings.Join(lines, "\n")
	}
	kept := lines[:maxLines]
	kept = append(kept, fmt.Sprintf("... (%d more lines)", len(lines)-maxLines))
	return strings.Join(kept, "\n")
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
