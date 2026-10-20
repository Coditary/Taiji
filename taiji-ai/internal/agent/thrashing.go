package agent

import (
	"fmt"
	"sort"
	"strings"
)

type thrashingState struct {
	failedEditsByPath      map[string]int
	lastTestFailSignature  string
	consecutiveTestFails   int
}

func newThrashingState() *thrashingState {
	return &thrashingState{failedEditsByPath: map[string]int{}}
}

func (s *thrashingState) observeToolOutcome(toolName, argsJSON, out string, hadError bool, threshold int) string {
	if s == nil || threshold <= 0 {
		return ""
	}
	switch strings.TrimSpace(toolName) {
	case "edit_file", "batch_edit":
		return s.observeEditOutcome(toolName, argsJSON, out, hadError, threshold)
	case "run_tests":
		return s.observeRunTestsOutcome(out, hadError, threshold)
	default:
		return ""
	}
}

func (s *thrashingState) observeEditOutcome(toolName, argsJSON, out string, hadError bool, threshold int) string {
	paths := editPathsForThrashing(toolName, argsJSON)
	if len(paths) == 0 {
		return ""
	}
	failed := hadError || editOutcomeFailed(out)
	if !failed {
		for _, p := range paths {
			s.failedEditsByPath[normalizePathKey(p)] = 0
		}
		return ""
	}
	var tripped string
	for _, p := range paths {
		key := normalizePathKey(p)
		s.failedEditsByPath[key]++
		if s.failedEditsByPath[key] >= threshold {
			tripped = p
		}
	}
	if tripped == "" {
		return ""
	}
	return circuitBreakerMessage(threshold, tripped)
}

func (s *thrashingState) observeRunTestsOutcome(out string, hadError bool, threshold int) string {
	if !hadError && strings.Contains(out, "✅ PASS") {
		s.consecutiveTestFails = 0
		s.lastTestFailSignature = ""
		return ""
	}
	sig := failedTestSignature(out)
	if sig == "" {
		if hadError {
			sig = "run_tests"
		} else {
			return ""
		}
	}
	if sig == s.lastTestFailSignature {
		s.consecutiveTestFails++
	} else {
		s.lastTestFailSignature = sig
		s.consecutiveTestFails = 1
	}
	if s.consecutiveTestFails < threshold {
		return ""
	}
	return circuitBreakerMessage(threshold, sig)
}

func circuitBreakerMessage(threshold int, target string) string {
	return fmt.Sprintf(
		"[RUNTIME CIRCUIT BREAKER: You have failed %d consecutive attempts on %s. STOP guessing. Either: (1) use file_outline / go_symbol_intel to re-verify types, (2) use workspace_checkpoint diff/restore to roll back to a clean state, or (3) call ask_user if requirements are ambiguous.]",
		threshold, target)
}

func editOutcomeFailed(out string) bool {
	if strings.Contains(out, "[DIAGNOSTICS WARNING]") {
		return true
	}
	lower := strings.ToLower(out)
	return strings.Contains(lower, "old_text not found") ||
		strings.Contains(lower, "old_text matched")
}

func editPathsForThrashing(toolName, argsJSON string) []string {
	if paths := batchEditPaths(argsJSON); len(paths) > 0 {
		return paths
	}
	if p := toolPath(toolName, argsJSON); p != "" {
		return []string{p}
	}
	return nil
}

func failedTestSignature(out string) string {
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "FAIL:") {
			names = append(names, line)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return strings.Join(names, "|")
}
