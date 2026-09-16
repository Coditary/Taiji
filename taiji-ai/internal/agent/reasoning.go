package agent

import (
	"context"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

const (
	planSystemInstruction = "You are an analytical planner. Create a detailed solution plan for the user's request. Break down the problem, identify constraints, outline your approach, and note potential pitfalls. You are not speaking to the user — prepare internal analytical notes only."

	planToAnswerTemplate = "Here is the user's question:\n\n%s\n\nHere are your analytical notes:\n\n%s\n\nBased on these notes, write the perfect answer for the user. Respond in the same language as the user's question."
)

func usesTaggedCoT(mode taijicfg.ChainOfThoughtMode) bool {
	return mode.OrDefault() != taijicfg.ChainOfThoughtOff
}

// shouldRunPlanPass reports whether to run the two-step planner workflow (Method 3).
func shouldRunPlanPass(goal string, mode taijicfg.ChainOfThoughtMode) bool {
	if mode.OrDefault() != taijicfg.ChainOfThoughtFull {
		return false
	}
	if userSkippedThinking(goal) {
		return false
	}
	if userRequestsThinking(goal) {
		return true
	}
	return needsDeepReasoning(goal)
}

// shouldUseThoughtChain enables multi-step auto-continue for chained small thoughts.
func shouldUseThoughtChain(goal string, mode taijicfg.ChainOfThoughtMode) bool {
	if !usesTaggedCoT(mode) {
		return false
	}
	if userSkippedThinking(goal) || isTrivialGoal(goal) {
		return false
	}
	if mode == taijicfg.ChainOfThoughtFull {
		return true
	}
	return needsDeepReasoning(goal) || userRequestsThinking(goal)
}

func chainContinuePrompt() string {
	return "Write the next single reasoning step only: one " + thinkingOpenTag + " block (1–4 sentences), then " + thinkingCloseTag + ". " +
		"Then either " + defaultContinueTrigger + " on its own line (more steps needed) OR the final answer. " +
		"Do not repeat prior steps. Do not output multiple think blocks."
}

func finalAnswerOnlyPrompt() string {
	return "Your reasoning is complete. Write ONLY the final, well-formatted answer for the user. " +
		"No " + thinkingOpenTag + " tags and no " + defaultContinueTrigger + "."
}

func chainOnePerMessageAppendix() string {
	return "CRITICAL — ONE THOUGHT PER MESSAGE: Each reply may contain AT MOST ONE " + thinkingOpenTag + "..." + thinkingCloseTag + " pair. " +
		"After it, output ONLY " + defaultContinueTrigger + " on its own line (more steps needed) OR the final answer. " +
		"Never output two or more think blocks in a single message. Never glue " + defaultContinueTrigger + " directly to " + thinkingOpenTag + "."
}

// explicitThinkingRequestAppendix is appended when the user asks the model to think.
func explicitThinkingRequestAppendix() string {
	return "The user explicitly asked you to think this through. Use CHAINED small " + thinkingOpenTag + " blocks (one step each, 1–4 sentences). " +
		"Separate steps with " + defaultContinueTrigger + ". Write the final answer only after the last thought, with no continue token."
}

func thinkingBlockRepairPrompt() string {
	return "You opened " + thinkingOpenTag + " but never closed it with " + thinkingCloseTag + ". " +
		"Do not repeat your reasoning. Write only the final, well-structured answer for the user now — no thinking tags."
}

func hasUnclosedThinkingBlock(text string) bool {
	normalized := normalizeThinkingTags(strings.TrimSpace(text))
	if normalized == "" {
		return false
	}
	opens := strings.Count(normalized, thinkingOpenTag)
	closes := strings.Count(normalized, thinkingCloseTag)
	return opens > closes
}

func userRequestsThinking(goal string) bool {
	lower := strings.ToLower(strings.TrimSpace(goal))
	for _, kw := range []string{
		"nachdenken", "nach denken", "gedanken machen", "mach dir gedanken", "denk nach",
		"denk drüber", "denk drueber", "denk darüber", "denk darueber", "überleg",
		"ueberleg", "bedenk",
		"think about", "think through", "think first", "reason about",
		"<|think", "<|tink", "chain of thought",
	} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func userSkippedThinking(goal string) bool {
	lower := strings.ToLower(strings.TrimSpace(goal))
	for _, kw := range []string{
		"ohne nachzudenken", "ohne zu überlegen", "ohne zu ueberlegen",
		"direkt antwort", "kurz und knapp", "keep it short", "just answer",
		"no thinking", "don't think", "dont think",
	} {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func isTrivialGoal(goal string) bool {
	lower := strings.TrimSpace(strings.ToLower(goal))
	lower = strings.TrimRight(lower, "!?.…")
	for _, kw := range []string{
		"hallo", "hello", "hi", "hey", "moin", "servus", "guten tag", "guten morgen", "guten abend",
		"danke", "thanks", "thank you", "dankeschön", "dankeschoen",
		"tschüss", "tschuesss", "bye", "ciao", "goodbye", "auf wiedersehen",
	} {
		if lower == kw {
			return true
		}
	}
	return false
}

// runPlanPass executes the planner step of the two-step workflow (Method 3).
func runPlanPass(
	ctx context.Context,
	p provider.Provider,
	messages []provider.Message,
	req RunRequest,
	step int,
	emit func(events.Event),
	metrics *runMetricsCollector,
) (string, error) {
	planMessages := append([]provider.Message{}, messages...)
	planMessages[0] = provider.Message{
		Role:    planMessages[0].Role,
		Content: planMessages[0].Content + "\n\n" + planSystemInstruction,
	}

	emit(events.Event{Type: events.TypeStep, Step: step, Text: "step 1 (planning)"})
	if req.StreamThoughtStart != nil {
		_ = req.StreamThoughtStart(1)
	}
	if metrics != nil {
		metrics.OnThinkingActivity()
	}

	var streamed strings.Builder
	resp, err := p.Complete(ctx, provider.CompletionRequest{
		Messages: planMessages,
		OnDelta: func(delta string) error {
			streamed.WriteString(delta)
			if metrics != nil && strings.TrimSpace(delta) != "" {
				metrics.OnThinkingActivity()
			}
			if req.StreamThinkDelta != nil {
				return req.StreamThinkDelta(delta)
			}
			return nil
		},
	})
	if err != nil {
		return "", err
	}
	if req.Session != nil {
		req.Session.AddUsage(resp.Usage, step)
	}
	if metrics != nil {
		metrics.AddUsage(resp.Usage)
		metrics.pauseThinking()
	}

	plan := strings.TrimSpace(resp.Content)
	if plan == "" {
		plan = strings.TrimSpace(streamed.String())
	}
	emit(events.Event{Type: events.TypeThought, Step: step, Text: plan, Data: map[string]any{"phase": "plan"}})
	return plan, nil
}

func withPlanContext(goal, plan string) string {
	return fmt.Sprintf(planToAnswerTemplate, goal, plan)
}
