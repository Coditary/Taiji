package agent

import (
	"context"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

const memoryReflectionSystemPrompt = `## Memory reflection (mandatory)

A tool or shell error occurred during this run and nothing was stored yet.

Before you finish, call exactly one memory tool:
- memory_revise — if an existing memory entry helped (set outcome to hit/success)
- memory_remember — if this was a new verified fix or pattern

Use the correct kind (error, pattern, anti_pattern, workflow, insight, preference).
Do not reply to the user in this turn — only call the memory tool.`

func runMemoryReflectionPass(
	ctx context.Context,
	p provider.Provider,
	req RunRequest,
	messages []provider.Message,
	allToolDefs []provider.ToolDef,
	tracker *memoryRunTracker,
	emit func(events.Event),
) ([]provider.Message, error) {
	if tracker == nil || !tracker.needsReflection() || req.Memory == nil || !req.Memory.Enabled() {
		return messages, nil
	}
	if req.Config != nil && !req.Config.ResolvedMemory().ReflectOnErrorOrDefault() {
		return messages, nil
	}

	memTools := filterMemoryToolDefs(allToolDefs)
	if len(memTools) == 0 {
		return messages, nil
	}

	working := append([]provider.Message{}, messages...)
	working = append(working,
		provider.Message{Role: provider.RoleSystem, Content: memoryReflectionSystemPrompt},
		provider.Message{Role: provider.RoleUser, Content: session.MemoryReflectionUserPrompt},
	)

	for step := 0; step < 3; step++ {
		resp, err := completeWithRetry(ctx, p, req, working, memTools, 900+step, false, nil, nil, nil)
		if err != nil {
			return messages, nil
		}
		if len(resp.ToolCalls) == 0 {
			break
		}

		working = append(working, provider.Message{
			Role:      provider.RoleAssistant,
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		})

		outcomes := executeToolCalls(ctx, req.Hooks, req.Tools, hookEnvFor(req, 900+step, req.Goal), resp.ToolCalls, working)
		for i, tc := range resp.ToolCalls {
			outcome := outcomes[i]
			out := outcome.result.Text
			if outcome.err != nil {
				out = "error: " + outcome.err.Error()
			}
			emit(events.Event{
				Type:     events.TypeToolCall,
				Step:     900 + step,
				ToolName: tc.Name,
				ToolID:   tc.ID,
				Args:     tc.Args,
			})
			emit(events.Event{
				Type:     events.TypeObservation,
				Step:     900 + step,
				ToolName: tc.Name,
				ToolID:   tc.ID,
				Text:     out,
			})
			if tc.Name == "memory_remember" {
				emit(events.Event{Type: events.TypeMemoryStored, Step: 900 + step, Text: out})
				tracker.rememberCalls++
			}
			if tc.Name == "memory_revise" {
				emit(events.Event{Type: events.TypeMemoryRevised, Step: 900 + step, Text: out})
				tracker.reviseCalls++
			}
			working = append(working, provider.Message{
				Role:       provider.RoleTool,
				Name:       tc.Name,
				ToolCallID: tc.ID,
				Content:    out,
			})
		}
		if tracker.storedMemory() {
			return working, nil
		}
	}
	return messages, nil
}

func filterMemoryToolDefs(all []provider.ToolDef) []provider.ToolDef {
	out := make([]provider.ToolDef, 0, 3)
	for _, d := range all {
		if isMemoryTool(d.Name) {
			out = append(out, d)
		}
	}
	return out
}
