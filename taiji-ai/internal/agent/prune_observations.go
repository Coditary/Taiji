package agent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/provider"
)

const (
	pruneStepAgeThreshold = 4
	pruneSizeThreshold    = 4000
	pruneHeadTailLines    = 15
	pruneSizeMarker       = "lines pruned from older step"
)

type pruneOptions struct {
	cacheFriendly bool
}

type toolMsgInfo struct {
	step     int
	toolName string
	args     string
}

// pruneStaleToolObservations compacts stale or oversized tool observations before a model call.
func pruneStaleToolObservations(messages []provider.Message, currentStep int, opts pruneOptions) []provider.Message {
	if len(messages) == 0 || currentStep <= 0 {
		return messages
	}
	sizePruneAllowed := !opts.cacheFriendly || (currentStep%pruneStepAgeThreshold == 0)
	meta := indexToolMessages(messages)
	out := append([]provider.Message{}, messages...)

	lastMutation := map[string]int{}
	lastRead := map[string]int{}

	for i, m := range out {
		if m.Role != provider.RoleTool {
			continue
		}
		info := meta[i]
		if info.step <= 0 {
			continue
		}
		path := toolPath(info.toolName, info.args)
		switch info.toolName {
		case "write_file", "edit_file":
			if !toolHadError(m.Content) {
				if paths := batchEditPaths(info.args); len(paths) > 0 {
					for _, p := range paths {
						lastMutation[normalizePathKey(p)] = info.step
					}
				} else if path != "" {
					lastMutation[normalizePathKey(path)] = info.step
				}
			}
		case "batch_edit":
			if !toolHadError(m.Content) {
				for _, p := range batchEditPaths(info.args) {
					lastMutation[normalizePathKey(p)] = info.step
				}
			}
		case "read_file":
			if path != "" {
				lastRead[normalizePathKey(path)] = info.step
			}
		}
	}

	for i, m := range out {
		if m.Role != provider.RoleTool {
			continue
		}
		info := meta[i]
		content := m.Content
		changed := false

		if info.toolName == "read_file" {
			path := normalizePathKey(toolPath(info.toolName, info.args))
			if path != "" {
				if mutStep := lastMutation[path]; mutStep > 0 && info.step < mutStep {
					content = staleReadPlaceholder(toolPath(info.toolName, info.args))
					changed = true
				} else if readStep := lastRead[path]; readStep > 0 && info.step < readStep {
					content = staleReadPlaceholder(toolPath(info.toolName, info.args))
					changed = true
				}
			}
		}

		if sizePruneAllowed && !changed && currentStep-info.step > pruneStepAgeThreshold && len(content) > pruneSizeThreshold {
			if info.toolName == "read_file" || info.toolName == "grep" || info.toolName == "shell" {
				if !strings.Contains(content, pruneSizeMarker) {
					content = pruneOversizedObservation(content)
					changed = true
				}
			}
		}
		if changed {
			out[i].Content = content
		}
	}
	return out
}

func indexToolMessages(messages []provider.Message) map[int]toolMsgInfo {
	meta := map[int]toolMsgInfo{}
	pending := map[string]toolMsgInfo{}
	step := 0
	for i, m := range messages {
		if m.Role == provider.RoleAssistant && len(m.ToolCalls) > 0 {
			step++
			for _, tc := range m.ToolCalls {
				pending[tc.ID] = toolMsgInfo{step: step, toolName: tc.Name, args: tc.Args}
			}
		}
		if m.Role == provider.RoleTool {
			info := pending[m.ToolCallID]
			if info.toolName == "" {
				info.toolName = m.Name
			}
			meta[i] = info
		}
	}
	return meta
}

func batchEditPaths(argsJSON string) []string {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" {
		return nil
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return nil
	}
	raw, ok := args["files"]
	if !ok {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return nil
	}
	var paths []string
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if p, ok := m["path"].(string); ok && strings.TrimSpace(p) != "" {
			paths = append(paths, strings.TrimSpace(p))
		}
	}
	return paths
}

func toolPath(toolName, argsJSON string) string {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	if p, ok := args["path"].(string); ok {
		return strings.TrimSpace(p)
	}
	return ""
}

func normalizePathKey(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	if abs, err := filepath.Abs(path); err == nil {
		return filepath.Clean(abs)
	}
	return filepath.Clean(path)
}

func staleReadPlaceholder(path string) string {
	return "[Pruned stale read_file output for " + path + " (superseded by later edit/read)]"
}

func pruneOversizedObservation(content string) string {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	if len(lines) <= pruneHeadTailLines*2 {
		return content
	}
	pruned := len(lines) - pruneHeadTailLines*2
	head := strings.Join(lines[:pruneHeadTailLines], "\n")
	tail := strings.Join(lines[len(lines)-pruneHeadTailLines:], "\n")
	return head + "\n[... " + itoa(pruned) + " lines pruned from older step to save context ...]\n" + tail
}

func toolHadError(content string) bool {
	lower := strings.ToLower(strings.TrimSpace(content))
	return strings.HasPrefix(lower, "error:") || strings.Contains(lower, "(error:")
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}
