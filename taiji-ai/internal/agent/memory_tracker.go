package agent

import (
	"encoding/json"
	"strings"
)

type memoryRunTracker struct {
	toolErrors       int
	autoRecalls      int
	recallByKind     map[string]int
	rememberCalls    int
	reviseCalls      int
	lastRecalledID   string
}

func newMemoryRunTracker() *memoryRunTracker {
	return &memoryRunTracker{recallByKind: map[string]int{}}
}

func (t *memoryRunTracker) recordToolCall(name, args string, hadError, autoRecall bool) {
	if t == nil {
		return
	}
	name = strings.TrimSpace(name)
	if hadError && !isMemoryTool(name) {
		t.toolErrors++
	}
	if autoRecall {
		t.autoRecalls++
	}
	switch name {
	case "memory_recall":
		kind := strings.ToLower(strings.TrimSpace(parseStringArg(args, "kind")))
		if kind == "" {
			kind = "unspecified"
		}
		t.recallByKind[kind]++
	case "memory_remember":
		t.rememberCalls++
	case "memory_revise":
		t.reviseCalls++
	}
}

func (t *memoryRunTracker) setRecalledID(id string) {
	if t == nil {
		return
	}
	id = strings.TrimSpace(id)
	if id != "" {
		t.lastRecalledID = id
	}
}

func (t *memoryRunTracker) storedMemory() bool {
	if t == nil {
		return false
	}
	return t.rememberCalls > 0 || t.reviseCalls > 0
}

func (t *memoryRunTracker) needsReflection() bool {
	if t == nil {
		return false
	}
	return t.toolErrors > 0 && !t.storedMemory()
}

func isMemoryTool(name string) bool {
	switch strings.TrimSpace(name) {
	case "memory_recall", "memory_remember", "memory_revise":
		return true
	default:
		return false
	}
}

func parseStringArg(argsJSON, key string) string {
	argsJSON = strings.TrimSpace(argsJSON)
	if argsJSON == "" {
		return ""
	}
	var args map[string]any
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return ""
	}
	v, _ := args[key].(string)
	return v
}
