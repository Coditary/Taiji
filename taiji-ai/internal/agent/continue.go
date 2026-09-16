package agent

import (
	"context"
	"fmt"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
)

const defaultContinueTrigger = "<|continue|>"

type continueConfig struct {
	trigger string
	max     int
	prompt  string
}

type completeOptions struct {
	streamContentAsThinking bool
	allowContinue           bool
	useTaggedCoT            bool
}

func resolveContinueConfig(cfg *taijicfg.Config) continueConfig {
	out := continueConfig{
		trigger: defaultContinueTrigger,
		max:     8,
		prompt:  chainContinuePrompt(),
	}
	if cfg == nil {
		return out
	}
	if cfg.Agent.ContinueTrigger != "" {
		out.trigger = cfg.Agent.ContinueTrigger
	}
	if cfg.Agent.ContinueTrigger == "off" || cfg.Agent.ContinueTrigger == "false" {
		out.trigger = ""
	}
	if cfg.Agent.MaxContinues > 0 {
		out.max = cfg.Agent.MaxContinues
	}
	if strings.TrimSpace(cfg.Agent.ContinuePrompt) != "" {
		out.prompt = strings.TrimSpace(cfg.Agent.ContinuePrompt)
	}
	return out
}

func responseRequestsContinue(resp *provider.CompletionResponse, trigger string) bool {
	if trigger == "" || resp == nil {
		return false
	}
	if containsContinueTrigger(resp.Content, trigger) {
		return true
	}
	for _, block := range resp.ThinkingBlocks {
		if containsContinueTrigger(block.Text, trigger) {
			return true
		}
	}
	return false
}

func containsContinueTrigger(text, trigger string) bool {
	text = strings.TrimSpace(text)
	if text == "" {
		return false
	}
	if strings.Contains(text, "<|continue") {
		return true
	}
	if strings.HasSuffix(text, trigger) {
		return true
	}
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == trigger {
			return true
		}
	}
	return false
}

func stripContinueTrigger(text, trigger string) string {
	if trigger == "" {
		return text
	}
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.TrimSpace(line) == trigger {
			continue
		}
		out = append(out, line)
	}
	return strings.TrimRight(strings.Join(out, "\n"), "\n")
}

func completeWithRetry(
	ctx context.Context,
	p provider.Provider,
	req RunRequest,
	working []provider.Message,
	toolDefs []provider.ToolDef,
	step int,
	toolNudged bool,
	onDelta func(string) error,
	onThink func(string) error,
) (*provider.CompletionResponse, error) {
	compReq := buildCompletionRequest(req, p, working, toolDefs, step, toolNudged, onDelta, onThink)
	resp, err := p.Complete(ctx, compReq)
	if err == nil {
		return resp, nil
	}
	if !strings.Contains(err.Error(), "empty response from model") {
		return nil, err
	}
	if compReq.ToolChoice == "required" {
		compReq.ToolChoice = "auto"
		compReq.Tools = toolDefs
		resp, err = p.Complete(ctx, compReq)
		if err == nil {
			return resp, nil
		}
	}
	return p.Complete(ctx, compReq)
}

func completeWithContinues(
	ctx context.Context,
	p provider.Provider,
	messages []provider.Message,
	toolDefs []provider.ToolDef,
	req RunRequest,
	step int,
	cfg continueConfig,
	opts completeOptions,
	emit func(events.Event),
	metrics *runMetricsCollector,
) ([]provider.Message, *provider.CompletionResponse, error) {
	working := append([]provider.Message{}, messages...)
	allowContinue := opts.allowContinue && cfg.trigger != ""

	var last *provider.CompletionResponse
	totalThoughts := 0
	toolNudged := false
	for cont := 0; cont <= cfg.max; cont++ {
		if cont > 0 {
			if !allowContinue {
				break
			}
			emit(events.Event{
				Type: events.TypeContinue,
				Step: step,
				Text: fmt.Sprintf("auto-continue %d", cont),
			})
		}

		var streamed strings.Builder
		var thinkBuf strings.Builder
		var tagParser *thinkingStreamParser

		streamDelta := req.StreamDelta
		if metrics != nil && streamDelta != nil {
			orig := streamDelta
			streamDelta = func(delta string) error {
				if strings.TrimSpace(delta) != "" {
					metrics.OnAnswerDelta()
				}
				return orig(delta)
			}
		}

		streamThink := req.StreamThinkDelta
		if metrics != nil {
			orig := streamThink
			streamThink = func(delta string) error {
				if strings.TrimSpace(delta) != "" {
					metrics.OnThinkingActivity()
				}
				if orig != nil {
					return orig(delta)
				}
				return nil
			}
		}

		streamThoughtStart := req.StreamThoughtStart
		if metrics != nil {
			orig := streamThoughtStart
			streamThoughtStart = func(n int) error {
				metrics.OnThinkingActivity()
				if orig != nil {
					return orig(n)
				}
				return nil
			}
		}

		useTagsThisPass := opts.useTaggedCoT
		onAnswerDelta := streamDelta
		if useTagsThisPass && streamDelta != nil {
			tagParser = newThinkingStreamParser(totalThoughts, allowContinue, streamThoughtStart, streamThink, streamDelta)
			onAnswerDelta = tagParser.Write
		}

		thoughtStarted := false
		onThinkDelta := func(delta string) error {
			thinkBuf.WriteString(delta)
			if metrics != nil && strings.TrimSpace(delta) != "" {
				metrics.OnThinkingActivity()
			}
			// Tagged chain-of-thought uses  blocks in the content stream.
			if useTagsThisPass {
				return nil
			}
			if req.StreamThinkDelta == nil && req.StreamThoughtStart == nil {
				return nil
			}
			if !thoughtStarted {
				thoughtStarted = true
				if req.StreamThoughtStart != nil {
					if err := req.StreamThoughtStart(totalThoughts + 1); err != nil {
						return err
					}
				}
			}
			if req.StreamThinkDelta != nil {
				return req.StreamThinkDelta(delta)
			}
			return nil
		}

		onContentDelta := func(delta string) error {
			streamed.WriteString(delta)
			if onAnswerDelta != nil {
				return onAnswerDelta(delta)
			}
			return nil
		}

		var onDelta func(string) error
		if req.StreamDelta != nil {
			onDelta = onContentDelta
		}
		var onThink func(string) error
		if req.StreamThinkDelta != nil {
			onThink = onThinkDelta
		}
		resp, err := completeWithRetry(ctx, p, req, working, toolDefs, step, toolNudged, onDelta, onThink)
		if err != nil {
			return working, nil, err
		}
		if req.Session != nil {
			req.Session.AddUsage(resp.Usage, step)
		}
		if metrics != nil {
			metrics.AddUsage(resp.Usage)
		}
		if tagParser != nil {
			if err := tagParser.Flush(); err != nil {
				return working, nil, err
			}
		}

		emitCompletedThinking(emit, step, resp, streamed.String(), thinkBuf.String(), useTagsThisPass, req)
		last = resp

		if strings.TrimSpace(resp.Content) != "" && req.StreamDelta == nil {
			emit(events.Event{Type: events.TypeMessage, Step: step, Text: extractAnswerText(resp, streamed.String(), useTagsThisPass)})
		}

		if len(resp.ToolCalls) > 0 {
			working = append(working, provider.Message{
				Role:      provider.RoleAssistant,
				Content:   resp.Content,
				ToolCalls: append([]provider.ToolCall(nil), resp.ToolCalls...),
			})
			return working, resp, nil
		}

		content := strings.TrimSpace(resp.Content)
		if content == "" {
			content = strings.TrimSpace(streamed.String())
		}

		if len(toolDefs) > 0 && !toolNudged &&
			(goalExpectsTools(req.Goal) || respDescribesToolUse(content)) {
			toolNudged = true
			working = append(working,
				provider.Message{Role: provider.RoleAssistant, Content: content},
				provider.Message{Role: provider.RoleUser, Content: toolUseNudge},
			)
			continue
		}

		blocks := parseAllThinkingBlocks(content)
		if allowContinue {
			if len(blocks) > 0 {
				totalThoughts++
			}
		} else {
			totalThoughts += len(blocks)
		}

		if useTagsThisPass && hasUnclosedThinkingBlock(content) {
			onRepairDelta := func(delta string) error {
				if req.StreamDelta != nil {
					if err := req.StreamDelta(delta); err != nil {
						return err
					}
				}
				if metrics != nil && strings.TrimSpace(delta) != "" {
					metrics.OnAnswerDelta()
				}
				return nil
			}
			repaired, repairResp, err := repairUnclosedThinking(ctx, p, working, toolDefs, req, content, onRepairDelta)
			if err != nil {
				return working, nil, err
			}
			if repairResp != nil {
				answer := strings.TrimSpace(extractAnswerText(repairResp, repaired, false))
				if answer == "" {
					answer = strings.TrimSpace(repaired)
				}
				working = append(working,
					provider.Message{Role: provider.RoleAssistant, Content: content},
					provider.Message{Role: provider.RoleUser, Content: thinkingBlockRepairPrompt()},
					provider.Message{Role: provider.RoleAssistant, Content: answer},
				)
				repairResp.Content = answer
				if req.Session != nil {
					req.Session.AddUsage(repairResp.Usage, step)
				}
				if metrics != nil {
					metrics.AddUsage(repairResp.Usage)
				}
				if strings.TrimSpace(answer) != "" && req.StreamDelta == nil {
					emit(events.Event{Type: events.TypeMessage, Step: step, Text: answer})
				}
				return working, repairResp, nil
			}
		}

		answer := extractAnswerText(resp, streamed.String(), useTagsThisPass)
		tailAnswer := strings.TrimSpace(extractAnswerFromChained(content))
		hasContinue := containsContinueTrigger(content, cfg.trigger)
		needsMore := hasContinue ||
			len(blocks) > 1 ||
			(len(blocks) >= 1 && tailAnswer != "" && isChainNoise(tailAnswer))
		if !allowContinue || !needsMore || cont >= cfg.max {
			if answer == "" || isChainNoise(answer) {
				answer = extractAnswerFromChained(content)
			}
			working = append(working, provider.Message{Role: provider.RoleAssistant, Content: answer})
			resp.Content = answer
			return working, resp, nil
		}

		assistantTurn := truncateToFirstChainStep(content, cfg.trigger)
		userPrompt := cfg.prompt
		if cont+1 >= cfg.max {
			userPrompt = finalAnswerOnlyPrompt()
		}
		working = append(working,
			provider.Message{Role: provider.RoleAssistant, Content: assistantTurn},
			provider.Message{Role: provider.RoleUser, Content: userPrompt},
		)
	}

	return working, last, nil
}

func repairUnclosedThinking(
	ctx context.Context,
	p provider.Provider,
	working []provider.Message,
	toolDefs []provider.ToolDef,
	req RunRequest,
	partialContent string,
	onContentDelta func(string) error,
) (string, *provider.CompletionResponse, error) {
	repairMessages := append(append([]provider.Message{}, working...),
		provider.Message{Role: provider.RoleAssistant, Content: partialContent},
		provider.Message{Role: provider.RoleUser, Content: thinkingBlockRepairPrompt()},
	)
	var buf strings.Builder
	var onDelta func(string) error
	if req.StreamDelta != nil {
		onDelta = func(delta string) error {
			buf.WriteString(delta)
			return onContentDelta(delta)
		}
	}
	resp, err := p.Complete(ctx, buildCompletionRequest(req, p, repairMessages, toolDefs, 1, false, onDelta, nil))
	if err != nil {
		return "", nil, err
	}
	streamed := buf.String()
	if strings.TrimSpace(resp.Content) == "" {
		resp.Content = streamed
	}
	return streamed, resp, nil
}

func emitCompletedThinking(
	emit func(events.Event),
	step int,
	resp *provider.CompletionResponse,
	streamed string,
	nativeThink string,
	useTags bool,
	req RunRequest,
) {
	if len(resp.ThinkingBlocks) > 0 {
		for idx, block := range resp.ThinkingBlocks {
			text := strings.TrimSpace(block.Text)
			if text == "" {
				continue
			}
			ev := events.Event{Type: events.TypeThought, Step: step, Text: text}
			ev.Data = map[string]any{"thinking_index": idx}
			if block.Source != "" {
				ev.Data["thinking_source"] = block.Source
			}
			emit(ev)
		}
		return
	}

	content := strings.TrimSpace(resp.Content)
	if content == "" {
		content = strings.TrimSpace(streamed)
	}
	if useTags {
		blocks := parseAllThinkingBlocks(content)
		for idx, block := range blocks {
			if block == "" {
				continue
			}
			ev := events.Event{Type: events.TypeThought, Step: step, Text: block}
			ev.Data = map[string]any{"thinking_index": idx}
			emit(ev)
		}
		if len(blocks) > 0 {
			return
		}
	}
	if nativeThink != "" {
		emit(events.Event{Type: events.TypeThought, Step: step, Text: nativeThink})
	}
}
