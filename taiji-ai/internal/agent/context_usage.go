package agent

import (
	"encoding/json"
	"strings"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/decisions"
	"github.com/coditary/taiji/internal/memory"
	"github.com/coditary/taiji/internal/rag"
	"github.com/coditary/taiji/internal/router"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
	"github.com/coditary/taiji/internal/tools/mcp"
)

// EstimateContextUsage approximates per-category context usage for the TUI breakdown.
func EstimateContextUsage(req RunRequest, activeSkillNames []string) session.ContextUsageBreakdown {
	agentRT := session.AgentRuntime{}
	if req.Session != nil {
		agentRT = req.Session.AgentRuntimeForRun(req.Config)
	} else if req.Config != nil {
		agentRT = (&session.Session{}).AgentRuntimeForRun(req.Config)
	}

	core, mcpExtra, skillsExtra := systemPromptSections(req, agentRT, activeSkillNames)

	conversation := 0
	if req.Session != nil {
		conversation = session.EstimateMessagesTokens(req.Session.State.Messages)
	}

	toolsTokens := 0
	if req.Tools != nil {
		defs := filterProviderTools(req.Tools, agentRT)
		if raw, err := json.Marshal(defs); err == nil {
			toolsTokens = session.EstimateTokens(string(raw))
			if len(defs) > 0 {
				toolsTokens += len(defs) * 12
			}
		}
	}
	mcpTokens := 0
	if mcpExtra != "" {
		mcpTokens = session.EstimateTokens(mcpExtra)
	}

	rulesTokens := 0
	if appendix, err := loadAgentsAppendix(projectWorkspaceRoot(req)); err == nil && appendix != "" {
		rulesTokens = session.EstimateTokens(appendix)
	}

	skillsTokens := session.EstimateTokens(skillsExtra)
	systemTokens := session.EstimateTokens(core)
	if systemTokens > 0 {
		systemTokens += 4
	}
	if rulesTokens > 0 && systemTokens >= rulesTokens {
		systemTokens -= rulesTokens
	}

	return session.ContextUsageBreakdown{
		SystemTokens:       systemTokens,
		ToolsTokens:        toolsTokens,
		MCPTokens:          mcpTokens,
		RulesTokens:        rulesTokens,
		SkillsTokens:       skillsTokens,
		ConversationTokens: conversation,
	}
}

func systemPromptSections(req RunRequest, agentRT session.AgentRuntime, activeSkillNames []string) (core string, mcpExtra string, skillsExtra string) {
	cotMode := agentRT.ChainOfThought
	systemPrompt := taijicfg.DefaultSystemPrompt()
	if req.Config != nil {
		if sp := strings.TrimSpace(req.Config.Agent.SystemPrompt); sp != "" {
			systemPrompt = sp
		} else if usesTaggedCoT(cotMode) {
			systemPrompt = taijicfg.DefaultTaggedCoTSystemPrompt()
		}
		if appendix := cotMode.SystemPromptAppendix(); appendix != "" {
			systemPrompt += appendix
		}
	}
	if extra := strings.TrimSpace(agentRT.SystemPromptExtra); extra != "" {
		systemPrompt += "\n\n" + extra
	}
	if preset := strings.TrimSpace(sessionPresentationPreset(req.Session)); preset != "" {
		if appendix := presentationPromptAppendix(preset); appendix != "" {
			systemPrompt += "\n\n" + appendix
		}
	}
	if appendix, err := loadAgentsAppendix(projectWorkspaceRoot(req)); err == nil && appendix != "" {
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
	if agentRT.DecisionsEnabled && req.DecisionProfiles != nil {
		if appendix, err := decisions.SystemPromptAppendix(req.DecisionProfiles); err == nil && appendix != "" {
			systemPrompt += "\n\n" + appendix
		}
	}

	core = systemPrompt

	if appendix := mcp.SystemPromptAppendix(req.MCP); appendix != "" {
		mcpExtra = appendix
	}

	if req.Skills != nil {
		var parts []string
		if appendix, err := skills.SystemPromptAppendix(req.Skills); err == nil && appendix != "" {
			parts = append(parts, appendix)
		}
		names := activeSkillNames
		if len(names) == 0 {
			names = req.SkillNames
		}
		if len(names) > 0 {
			all, err := req.Skills.LoadAll()
			if err == nil {
				if appendix := skills.BuildSystemAppendix(all, names); appendix != "" {
					parts = append(parts, appendix)
				}
			}
		}
		skillsExtra = strings.Join(parts, "\n\n")
	}

	return core, mcpExtra, skillsExtra
}
