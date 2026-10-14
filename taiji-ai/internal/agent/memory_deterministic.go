package agent

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/memory"
)

func runDeterministicMemoryRevision(
	ctx context.Context,
	req RunRequest,
	tracker *memoryRunTracker,
	finalAnswer string,
	emit func(events.Event),
) {
	if tracker == nil || req.Memory == nil || !req.Memory.Enabled() {
		return
	}
	if tracker.autoRecalls == 0 {
		return
	}
	id := strings.TrimSpace(tracker.lastRecalledID)
	if id == "" {
		return
	}
	if tracker.storedMemory() {
		return
	}
	if strings.TrimSpace(finalAnswer) == "" {
		return
	}
	_, err := req.Memory.Revise(ctx, memory.ReviseRequest{
		ID:           id,
		RevisionNote: "Auto-revised after successful run using recalled memory",
		Outcome:      "hit",
	})
	if err != nil {
		return
	}
	tracker.reviseCalls++
	text := "auto-revised id=" + id
	args, _ := json.Marshal(map[string]any{"id": id, "outcome": "hit"})
	emit(events.Event{
		Type:     events.TypeToolCall,
		Step:     900,
		ToolName: "memory_revise",
		Args:     string(args),
		Text:     "auto",
	})
	emit(events.Event{
		Type:       events.TypeObservation,
		Step:       900,
		ToolName:   "memory_revise",
		Text:       text,
		DurationMs: 0,
	})
}
