package agent

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/coditary/taiji/internal/agentsdoc"
	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools"
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
	Config             *taijicfg.Config
	RAG                *rag.Service
	MCP                *mcp.Bridge
	WorkDir            string
	EventHandler       func(events.Event)
	StreamDelta        func(delta string) error
	StreamThinkDelta   func(delta string) error
	StreamThoughtStart func(thoughtNum int) error
	StreamEnd          func() error
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
		providerID = req.Session.State.Provider
	}
	p, err := req.Providers.Get(providerID)
	if err != nil {
		p, err = req.Providers.Default()
		if err != nil {
			return nil, err
		}
		providerID = p.ID()
	}

	maxSteps := 25
	cotMode := taijicfg.ChainOfThoughtOff
	systemPrompt := taijicfg.DefaultSystemPrompt()
	if req.Config != nil {
		cotMode = req.Config.Agent.ChainOfThought.OrDefault()
		if req.Config.Agent.MaxSteps > 0 {
			maxSteps = req.Config.Agent.MaxSteps
		}
		if sp := strings.TrimSpace(req.Config.Agent.SystemPrompt); sp != "" {
			systemPrompt = sp
		} else if usesTaggedCoT(cotMode) {
			systemPrompt = taijicfg.DefaultTaggedCoTSystemPrompt()
		}
		if appendix := cotMode.SystemPromptAppendix(); appendix != "" {
			systemPrompt += appendix
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
	if appendix := mcp.SystemPromptAppendix(req.MCP); appendix != "" {
		systemPrompt += "\n\n" + appendix
	}
	if req.Skills != nil && len(req.SkillNames) > 0 {
		all, err := req.Skills.LoadAll()
		if err != nil {
			return nil, err
		}
		appendix := skills.BuildSystemAppendix(all, req.SkillNames)
		if appendix != "" {
			systemPrompt += "\n\n" + appendix
		}
	}

	messages := []provider.Message{{Role: provider.RoleSystem, Content: systemPrompt}}
	if req.Session != nil && len(req.Session.State.Messages) > 0 {
		messages = append(messages, req.Session.State.Messages...)
	}
	messages = append(messages, provider.Message{Role: provider.RoleUser, Content: req.Goal})

	toolDefs := toProviderTools(req.Tools.Definitions())
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
	for step := 1; step <= maxSteps; step++ {
		if !shouldRunPlanPass(req.Goal, cotMode) || step > 1 {
			emit(events.Event{Type: events.TypeStep, Step: step, Text: fmt.Sprintf("step %d", step)})
		}

		messages, resp, err = completeWithContinues(ctx, p, messages, toolDefs, req, step, continueCfg, completeOptions{
			useTaggedCoT:  tagCoT,
			allowContinue: shouldUseThoughtChain(req.Goal, cotMode),
		}, emit, metrics)
		if err != nil {
			emit(events.Event{Type: events.TypeError, Step: step, Text: err.Error()})
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

		if hasRecentToolObservation(messages) {
			finalAnswer = extractAnswerText(resp, resp.Content, tagCoT)
			if strings.TrimSpace(finalAnswer) == "" {
				finalAnswer = lastToolObservation(messages)
			}
			if req.StreamEnd != nil {
				_ = req.StreamEnd()
			}
			emit(events.Event{Type: events.TypeDone, Step: step, Text: finalAnswer})
			break
		}

		for _, tc := range resp.ToolCalls {
			if isDuplicateToolCall(messages, tc) {
				finalAnswer = lastToolObservation(messages)
				emit(events.Event{Type: events.TypeDone, Step: step, Text: finalAnswer})
				goto finishRun
			}
			mcpServer, isMCP := mcpServerOf(req.Tools, tc.Name)
			emit(events.Event{
				Type:      events.TypeToolCall,
				Step:      step,
				ToolName:  tc.Name,
				ToolID:    tc.ID,
				Args:      tc.Args,
				IsMCP:     isMCP,
				MCPServer: mcpServer,
			})
			toolStart := time.Now()
			result, err := req.Tools.Run(ctx, tc.Name, tc.Args)
			toolDurationMs := time.Since(toolStart).Milliseconds()
			out := result.Text
			if err != nil {
				out = "error: " + err.Error()
			}
			obs := events.Event{
				Type:       events.TypeObservation,
				Step:       step,
				ToolName:   tc.Name,
				ToolID:     tc.ID,
				Text:       out,
				DurationMs: toolDurationMs,
				IsMCP:      isMCP,
				MCPServer:  mcpServer,
			}
			if len(result.Data) > 0 && err == nil {
				obs.Data = result.Data
			}
			emit(obs)
			messages = append(messages, provider.Message{
				Role:       provider.RoleTool,
				Name:       tc.Name,
				ToolCallID: tc.ID,
				Content:    out,
			})
		}
		if step == maxSteps {
			return nil, fmt.Errorf("max steps (%d) reached", maxSteps)
		}
	}

finishRun:
	if req.Session != nil {
		req.Session.State.Messages = messages
		req.Session.State.Provider = providerID
		if err := req.Session.Save(); err != nil {
			return nil, err
		}
	}

	return &RunResult{
		Answer:   finalAnswer,
		Steps:    len(messages),
		Messages: messages,
		Metrics:  metrics.Finalize(finalAnswer),
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
