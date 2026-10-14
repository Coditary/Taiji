package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/provider"
)

type memoryPrefetchResult struct {
	stored      bool
	recalled    bool
	storePath   string
	recallBlock string
}

func runMemoryPrefetch(
	ctx context.Context,
	req RunRequest,
	goal string,
	emit func(events.Event),
) (string, memoryPrefetchResult) {
	if req.Memory == nil || !req.Memory.Enabled() {
		return goal, memoryPrefetchResult{}
	}
	intent := memory.AnalyzeUserMessage(req.Memory.KindCatalog(), goal)
	if intent == nil {
		return goal, memoryPrefetchResult{}
	}

	var result memoryPrefetchResult
	var blocks []string

	switch intent.Action {
	case memory.IntentStore:
		if intent.Store != nil {
			entry, err := req.Memory.Remember(ctx, *intent.Store)
			if err == nil {
				result.stored = true
				result.storePath = entry.Path
				blocks = append(blocks, fmt.Sprintf("[memory] Stored %s (%s) at %s", entry.Kind, entry.Title, entry.ID))
				emit(events.Event{
					Type:     events.TypeToolCall,
					Step:     1,
					ToolName: "memory_remember",
					Args:     prefetchRememberArgs(*intent.Store),
					Text:     "auto",
				})
				emit(events.Event{
					Type:       events.TypeObservation,
					Step:       1,
					ToolName:   "memory_remember",
					Text:       fmt.Sprintf("remembered id=%s kind=%s path=%s", entry.ID, entry.Kind, entry.Path),
					DurationMs: 0,
				})
			}
		}
	case memory.IntentRecall:
		var all []memory.Entry
		for _, kind := range intent.Kinds {
			entries, err := req.Memory.Recall(ctx, memory.RecallRequest{
				Query: intent.RecallQuery,
				Kind:  string(kind),
				TopK:  3,
			})
			if err != nil {
				continue
			}
			all = append(all, entries...)
		}
		recallArgs := prefetchRecallArgs(intent)
		emit(events.Event{
			Type:     events.TypeToolCall,
			Step:     1,
			ToolName: "memory_recall",
			Args:     recallArgs,
			Text:     "auto",
		})
		obsText := "no memory entries matched"
		if len(all) > 0 {
			result.recalled = true
			block := memory.FormatRecallResults(all)
			result.recallBlock = block
			obsText = block
			blocks = append(blocks, "[memory prefetch]\n"+block)
		} else {
			blocks = append(blocks, "[memory prefetch] no entries matched for kinds: "+kindNames(intent.Kinds))
		}
		emit(events.Event{
			Type:       events.TypeObservation,
			Step:       1,
			ToolName:   "memory_recall",
			Text:       obsText,
			DurationMs: 0,
		})
	}

	if len(blocks) == 0 {
		return goal, result
	}
	preface := strings.Join(blocks, "\n\n")
	return strings.TrimSpace(preface + "\n\n" + goal), result
}

func runMemoryStoreFallback(
	ctx context.Context,
	req RunRequest,
	goal string,
	tracker *memoryRunTracker,
	emit func(events.Event),
) {
	if tracker == nil || tracker.storedMemory() {
		return
	}
	if req.Memory == nil || !req.Memory.Enabled() {
		return
	}
	intent := memory.AnalyzeUserMessage(req.Memory.KindCatalog(), goal)
	if intent == nil || intent.Action != memory.IntentStore || intent.Store == nil {
		return
	}
	entry, err := req.Memory.Remember(ctx, *intent.Store)
	if err != nil {
		return
	}
	tracker.rememberCalls++
	emit(events.Event{
		Type:       events.TypeToolCall,
		Step:       1,
		ToolName:   "memory_remember",
		Args:       prefetchRememberArgs(*intent.Store),
		Text:       "auto",
	})
	emit(events.Event{
		Type:       events.TypeObservation,
		Step:       1,
		ToolName:   "memory_remember",
		Text:       fmt.Sprintf("auto-remembered id=%s kind=%s path=%s", entry.ID, entry.Kind, entry.Path),
		DurationMs: 0,
	})
}

func prefetchRecallArgs(intent *memory.UserIntent) string {
	if intent == nil {
		return `{"auto":true}`
	}
	return fmt.Sprintf(`{"kind":%q,"query":%q,"auto":true}`,
		kindNames(intent.Kinds), intent.RecallQuery)
}

func prefetchRememberArgs(store memory.RememberRequest) string {
	return fmt.Sprintf(`{"kind":%q,"title":%q,"recall_text":%q,"auto":true}`,
		store.Kind, store.Title, store.RecallText)
}

func kindNames(kinds []memory.Kind) string {
	names := make([]string, 0, len(kinds))
	for _, k := range kinds {
		names = append(names, string(k))
	}
	return strings.Join(names, ", ")
}

func injectMemoryPrefetch(messages []provider.Message, goal string) []provider.Message {
	if strings.TrimSpace(goal) == "" {
		return messages
	}
	out := make([]provider.Message, len(messages))
	copy(out, messages)
	for i := len(out) - 1; i >= 0; i-- {
		if out[i].Role == provider.RoleUser {
			out[i].Content = goal
			break
		}
	}
	return out
}
