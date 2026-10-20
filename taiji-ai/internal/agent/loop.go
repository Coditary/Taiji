package agent

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/coditary/taiji/internal/agentsdoc"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/router"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
	"github.com/coditary/taiji/internal/tools/builtin"
	"github.com/coditary/taiji/internal/tools/mcp"
)

// RunRequest configures a single agent run.
type RunRequest struct {
	Goal               string
	ProviderID         string
	SkillNames         []string
	Session            *session.Session
	Providers          *provider.Registry
	Tools              *tools.Registry
	Skills             *skills.Loader
	SkillsCatalog      *skills.Catalog
	Config             *taijicfg.Config
	RAG                *rag.Service
	Memory             *memory.Service
	MCP                *mcp.Bridge
	DecisionProfiles   *decisions.Loader
	Hooks              *hooks.Runner
	Questions          *question.Service
	WorkDir            string
	EventHandler       func(events.Event)
	StreamDelta        func(delta string) error
	StreamThinkDelta   func(delta string) error
	StreamThoughtStart func(thoughtNum int) error
	StreamEnd          func() error
	SubRun             SubRunConfig
}

// RunResult is the final agent output.
type RunResult struct {
	Answer   string
	Steps    int
	Messages []provider.Message
	Metrics  RunMetrics
}

// Run executes the agent loop until completion or max steps.
func Run(ctx context.Context, req RunRequest) (*RunResult, error) {
	metrics := newRunMetricsCollector()
	if req.Providers == nil {
		return nil, fmt.Errorf("provider registry is required")
	}
	if req.Tools == nil {
		return nil, fmt.Errorf("tool registry is required")
	}
	providerID := strings.TrimSpace(req.ProviderID)
	if providerID == "" && req.Session != nil {
		providerID = req.Session.ResolveProviderID(req.Config)
	}
	p, err := req.Providers.Get(providerID)
	if err != nil {
		p, err = req.Providers.Default()
		if err != nil {
			return nil, err
		}
		providerID = p.ID()
	}

	agentRT := session.AgentRuntime{}
	if req.Session != nil {
		agentRT = req.Session.AgentRuntimeForRun(req.Config)
	} else if req.Config != nil {
		agentRT = (&session.Session{}).AgentRuntimeForRun(req.Config)
	}
	maxSteps := agentRT.MaxSteps
	if req.SubRun.Enabled && req.SubRun.MaxSteps > 0 {
		maxSteps = req.SubRun.MaxSteps
	}
	cotMode := agentRT.ChainOfThought
	systemPrompt := taijicfg.DefaultSystemPrompt()
	if req.SubRun.Enabled {
		systemPrompt = strings.TrimSpace(req.SubRun.SystemPrompt)
		if systemPrompt == "" {
			systemPrompt = defaultSubAgentSystemPrompt
		}
	} else if req.Config != nil {
		if sp := strings.TrimSpace(req.Config.Agent.SystemPrompt); sp != "" {
			systemPrompt = sp
		} else if usesTaggedCoT(cotMode) {
			systemPrompt = taijicfg.DefaultTaggedCoTSystemPrompt()
		}
		if appendix := cotMode.SystemPromptAppendix(); appendix != "" {
			systemPrompt += appendix
		}
	}
	if !req.SubRun.Enabled {
		if extra := strings.TrimSpace(agentRT.SystemPromptExtra); extra != "" {
			systemPrompt += "\n\n" + extra
		}
		if preset := strings.TrimSpace(sessionPresentationPreset(req.Session)); preset != "" {
			if appendix := presentationPromptAppendix(preset); appendix != "" {
				systemPrompt += "\n\n" + appendix
			}
		}
		if usesTaggedCoT(cotMode) && userRequestsThinking(req.Goal) {
			systemPrompt += "\n\n" + explicitThinkingRequestAppendix()
		}
		if shouldUseThoughtChain(req.Goal, cotMode) {
			systemPrompt += "\n\n" + chainOnePerMessageAppendix()
		}
		if appendix, err := loadAgentsAppendix(req.WorkDir); err != nil {
			return nil, err
		} else if appendix != "" {
			systemPrompt += "\n\n" + appendix
		}
		if appendix := rag.SystemPromptAppendix(req.RAG); appendix != "" {
			systemPrompt += "\n\n" + appendix
		}
		if agentRT.MemoryEnabled {
			if appendix := memory.SystemPromptAppendix(req.Memory); appendix != "" {
				systemPrompt += "\n\n" + appendix
			}
		}
		systemPrompt += "\n\n" + router.SystemPromptAppendix()
		if appendix := mcp.SystemPromptAppendix(req.MCP); appendix != "" {
			systemPrompt += "\n\n" + appendix
		}
		if agentRT.DecisionsEnabled && req.DecisionProfiles != nil {
			if appendix, err := decisions.SystemPromptAppendix(req.DecisionProfiles); err != nil {
				return nil, err
			} else if appendix != "" {
				systemPrompt += "\n\n" + appendix
			}
		}
		if req.Skills != nil {
			if appendix, err := skills.SystemPromptAppendix(req.Skills); err != nil {
				return nil, err
			} else if appendix != "" {
				systemPrompt += "\n\n" + appendix
			}
			if len(req.SkillNames) > 0 {
				all, err := req.Skills.LoadAll()
				if err != nil {
					return nil, err
				}
				appendix := skills.BuildSystemAppendix(all, req.SkillNames)
				if appendix != "" {
					systemPrompt += "\n\n" + appendix
				}
			}
		}
	}

	goal := strings.TrimSpace(req.Goal)

	messages := []provider.Message{{Role: provider.RoleSystem, Content: systemPrompt}}
	if req.Session != nil && len(req.Session.State.Messages) > 0 {
		messages = append(messages, req.Session.State.Messages...)
	}

	toolDefs := filterProviderTools(req.Tools, agentRT)
	memTracker := newMemoryRunTracker()
	continueCfg := resolveContinueConfig(req.Config)
	tagCoT := usesTaggedCoT(cotMode)

	emit := func(ev events.Event) {
		if req.Session != nil {
			ev.SessionID = req.Session.State.ID
		}
		if req.EventHandler != nil {
			req.EventHandler(ev)
		}
		if req.Session != nil && req.Session.Events != nil {
			_ = req.Session.Events.Emit(ev)
		}
	}

	sessionID := ""
	if req.Session != nil {
		sessionID = req.Session.State.ID
	}
	ctx = session.WithRunContext(ctx, agentRT)
	ctx = builtin.WithToolSessionID(ctx, sessionID)

	if req.Questions != nil {
		ctx = question.AttachRun(ctx, question.RunConfig{
			SessionID: sessionID,
			OnAsk: func(r question.Request) {
				emit(events.Event{
					Type:      events.TypeUserPrompt,
					SessionID: sessionID,
					IsAskUser: true,
					Data: map[string]any{
						"request_id": r.ID,
						"questions":  r.Questions,
					},
				})
			},
		})
	}

	hookEnv := hookEnvFor(req, 0, goal)
	if !req.SubRun.SkipHooks && req.Hooks != nil {
		if err := req.Hooks.RunSession(ctx, hooks.EventSessionStart, hooks.SessionData{Env: hookEnv, Answer: goal}); err != nil {
			return nil, err
		}
		prompt, err := req.Hooks.RunPromptBefore(ctx, hooks.PromptData{Env: hookEnv, Text: goal})
		if err != nil {
			_ = req.Hooks.RunSession(ctx, hooks.EventSessionError, hooks.SessionData{Env: hookEnv, Error: err.Error()})
			return nil, err
		}
		goal = strings.TrimSpace(prompt.Text)
	}
	var routed turnRouterResult
	if req.SubRun.SkipTurnRouter {
		routed = turnRouterResult{goal: goal, plan: router.Plan{Action: router.ActionUnknown, Reason: "sub-agent run"}}
	} else {
		routed = runTurnRouter(ctx, req, goal, emit)
	}
	if routed.plan.Action == router.ActionStop {
		answer := routed.stopAnswer
		emit(events.Event{Type: events.TypeDone, Step: 1, Text: answer})
		if req.Session != nil {
			messages = append(messages,
				provider.Message{Role: provider.RoleUser, Content: goal},
				provider.Message{Role: provider.RoleAssistant, Content: answer},
			)
			req.Session.State.Messages = messages
			req.Session.State.Provider = providerID
			_ = req.Session.Save()
		}
		runMetrics := metrics.Finalize(answer)
		runMetrics.Steps = 1
		emit(events.Event{Type: events.TypeTurnMetrics, Data: turnMetricsEventData(runMetrics)})
		return &RunResult{Answer: answer, Steps: 1, Messages: messages, Metrics: runMetrics}, nil
	}
	planAppendix := router.PlanAppendix(routed.plan)
	skillAppendix := skillPrefetchAppendix(routed.skillMatches)
	if planAppendix != "" {
		systemPrompt += "\n\n" + planAppendix
	}
	if skillAppendix != "" {
		systemPrompt += "\n\n" + skillAppendix
	}
	if !req.SubRun.Enabled {
		if todoAppendix := todoPlanAppendix(sessionID); todoAppendix != "" {
			systemPrompt += "\n\n" + todoAppendix
		}
	}
	messages[0].Content = systemPrompt
	if routed.prefetch.stored {
		memTracker.rememberCalls++
	}
	if routed.prefetch.recalled {
		memTracker.autoRecalls++
	}
	messages = append(messages, provider.Message{Role: provider.RoleUser, Content: routed.goal})

	if shouldRunPlanPass(req.Goal, cotMode) {
		plan, err := runPlanPass(ctx, p, messages, req, 1, emit, metrics)
		if err != nil {
			emit(events.Event{Type: events.TypeError, Step: 1, Text: err.Error()})
			return nil, err
		}
		messages = append(messages,
			provider.Message{Role: provider.RoleUser, Content: withPlanContext(req.Goal, plan)},
		)
	}

	var finalAnswer string
	var resp *provider.CompletionResponse
	var lastStep int
	thrashing := newThrashingState()
	thrashingThreshold := 3
	cacheFriendlyPrune := true
	if req.Config != nil {
		thrashingThreshold = req.Config.Agent.ThrashingThresholdOrDefault()
		cacheFriendlyPrune = req.Config.Agent.CacheFriendlyPruningOrDefault()
	}
	for step := 1; step <= maxSteps; step++ {
		lastStep = step
		if !shouldRunPlanPass(req.Goal, cotMode) || step > 1 {
			emit(events.Event{Type: events.TypeStep, Step: step, Text: fmt.Sprintf("step %d", step)})
		}

		stepMessages := messages
		if !req.SubRun.Enabled {
			if rem := todoPlanAppendix(sessionID); rem != "" {
				stepMessages = appendHookContext(messages, rem)
			}
		}
		pruneObs := true
		if req.Config != nil {
			pruneObs = req.Config.Agent.PruneStaleObservationsOrDefault()
		}
		if pruneObs && !req.SubRun.Enabled {
			messages = pruneStaleToolObservations(messages, step, pruneOptions{cacheFriendly: cacheFriendlyPrune})
		}

		if req.Hooks != nil {
			ctxData, hookErr := req.Hooks.RunContextBefore(ctx, hooks.ContextData{Env: hookEnvFor(req, step, goal)})
			if hookErr != nil {
				emit(events.Event{Type: events.TypeError, Step: step, Text: hookErr.Error()})
				runSessionErrorHook(ctx, req, hookEnvFor(req, step, goal), hookErr)
				return nil, hookErr
			}
			stepMessages = appendHookContext(messages, ctxData.AdditionalSystem)
		}

		messages, resp, err = completeWithContinues(ctx, p, stepMessages, toolDefs, req, step, continueCfg, completeOptions{
			useTaggedCoT:  tagCoT,
			allowContinue: shouldUseThoughtChain(req.Goal, cotMode),
		}, emit, metrics)
		if err != nil {
			emit(events.Event{Type: events.TypeError, Step: step, Text: err.Error()})
			runSessionErrorHook(ctx, req, hookEnvFor(req, step, goal), err)
			return nil, err
		}

		if len(resp.ToolCalls) == 0 {
			finalAnswer = extractAnswerText(resp, resp.Content, tagCoT)
			if req.StreamEnd != nil {
				_ = req.StreamEnd()
			}
			emit(events.Event{Type: events.TypeDone, Step: step, Text: finalAnswer})
			break
		}

		var skillObservations []string
		outcomes := executeToolCalls(ctx, req.Hooks, req.Tools, hookEnvFor(req, step, goal), resp.ToolCalls, messages)
		for i, tc := range resp.ToolCalls {
			outcome := outcomes[i]
			metrics.AddToolExecution(outcome.durationMs)
			mcpServer, isMCP := mcpServerOf(req.Tools, tc.Name)
			isSkill := tc.Name == skills.InvokeSkillToolName
			out := outcome.text
			if !outcome.duplicate {
				out = outcome.result.Text
				if outcome.err != nil {
					out = "error: " + outcome.err.Error()
				}
			}
			hadError := outcome.err != nil
			if !hadError && !isMemoryTool(tc.Name) {
				lower := strings.ToLower(strings.TrimSpace(out))
				hadError = strings.HasPrefix(lower, "error:") || strings.Contains(lower, "(error:")
			}
			autoRecall := false
			if req.Memory != nil && shouldAutoRecallMemory(tc.Name, out, outcome.err) {
				if hint, entryID := req.Memory.FormatErrorHintWithID(ctx, tc.Name, out); hint != "" {
					out += "\n\n" + hint
					autoRecall = true
					memTracker.setRecalledID(entryID)
					emit(events.Event{
						Type:     events.TypeMemoryRecalled,
						Step:     step,
						ToolName: tc.Name,
						Text:     hint,
					})
				}
			}
			memTracker.recordToolCall(tc.Name, tc.Args, hadError, autoRecall)
			if hadError && !req.SubRun.Enabled {
				if rem := todoPlanAppendix(sessionID); rem != "" {
					out += "\n\n" + rem
				}
			}
			if !req.SubRun.Enabled {
				if breaker := thrashing.observeToolOutcome(tc.Name, tc.Args, out, hadError, thrashingThreshold); breaker != "" {
					out += "\n\n" + breaker
				}
			}
			if isSkill {
				skillObservations = append(skillObservations, emitSkillToolActivity(emit, step, tc, outcome, mcpServer, isMCP)...)
			} else if isDecisionTool(tc.Name) {
				emitDecisionToolActivity(emit, step, tc, outcome)
			} else if isAskUserTool(tc.Name) {
				emitAskUserToolActivity(emit, step, sessionID, tc, outcome)
			} else {
				emit(events.Event{
					Type:      events.TypeToolCall,
					Step:      step,
					ToolName:  tc.Name,
					ToolID:    tc.ID,
					Args:      tc.Args,
					IsMCP:     isMCP,
					MCPServer: mcpServer,
				})
				emit(events.Event{
					Type:       events.TypeObservation,
					Step:       step,
					ToolName:   tc.Name,
					ToolID:     tc.ID,
					Text:       out,
					DurationMs: outcome.durationMs,
					IsMCP:      isMCP,
					MCPServer:  mcpServer,
				})
			}
			messages = append(messages, provider.Message{
				Role:       provider.RoleTool,
				Name:       tc.Name,
				ToolCallID: tc.ID,
				Content:    out,
			})
		}
		if len(skillObservations) > 0 {
			messages = appendSkillContinuation(messages)
		}
		if step == maxSteps {
			return nil, fmt.Errorf("max steps (%d) reached", maxSteps)
		}
	}

	if req.Memory != nil && !req.SubRun.SkipMemoryPasses {
		runDeterministicMemoryRevision(ctx, req, memTracker, finalAnswer, emit)
		if !memTracker.storedMemory() {
			messages, _ = runMemoryReflectionPass(ctx, p, req, messages, toolDefs, memTracker, emit)
		}
	}

	if req.Session != nil {
		req.Session.State.Messages = messages
		req.Session.State.Provider = providerID
		if err := req.Session.Save(); err != nil {
			return nil, err
		}
	}

	if req.Hooks != nil {
		_ = req.Hooks.RunSession(ctx, hooks.EventSessionIdle, hooks.SessionData{
			Env:    hookEnvFor(req, 0, goal),
			Answer: finalAnswer,
		})
	}

	runMetrics := metrics.Finalize(finalAnswer)
	runMetrics.Steps = lastStep
	if req.Session != nil {
		snap := req.Session.ContextSnapshot(req.Config, providerID)
		runMetrics.ContextTokens = snap.ContextTokens
		runMetrics.ContextWindow = snap.ContextWindow
		runMetrics.ContextUsedPct = snap.ContextUsedPct
	}
	emit(events.Event{
		Type: events.TypeTurnMetrics,
		Data: turnMetricsEventData(runMetrics),
	})

	return &RunResult{
		Answer:   finalAnswer,
		Steps:    len(messages),
		Messages: messages,
		Metrics:  runMetrics,
	}, nil
}

// mcpServerOf reports whether the named tool is backed by an MCP server.
func mcpServerOf(reg *tools.Registry, name string) (string, bool) {
	if reg == nil {
		return "", false
	}
	t, ok := reg.Get(name)
	if !ok {
		return "", false
	}
	if st, ok := t.(mcp.ServerTool); ok {
		return st.MCPServerName(), true
	}
	return "", false
}

func toProviderTools(defs []tools.Definition) []provider.ToolDef {
	out := make([]provider.ToolDef, 0, len(defs))
	for _, d := range defs {
		out = append(out, provider.ToolDef{
			Name:        d.Name,
			Description: d.Description,
			Parameters:  d.Parameters,
		})
	}
	return out
}

func loadAgentsAppendix(workDir string) (string, error) {
	dir := strings.TrimSpace(workDir)
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	content, found, err := agentsdoc.Load(dir)
	if err != nil || !found {
		return "", err
	}
	return agentsdoc.SystemAppendix(content), nil
}

func shouldAutoRecallMemory(toolName, out string, toolErr error) bool {
	if toolErr != nil {
		return true
	}
	switch toolName {
	case "memory_recall", "memory_remember", "memory_revise":
		return false
	}
	lower := strings.ToLower(strings.TrimSpace(out))
	if strings.HasPrefix(lower, "error:") {
		return true
	}
	return strings.Contains(lower, "(error:") || strings.Contains(lower, "\n(error:")
}
