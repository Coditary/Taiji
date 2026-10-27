package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/agent"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/skills"
)

func (h *runtimeHandle) runCouncilChat(
	ctx context.Context,
	goalText string,
	providerID string,
	actors []string,
	rt *runtime.Runtime,
	sendDelta func(string) error,
	sendEvent func(events.Event),
) (*agent.RunResult, error) {
	if h == nil || h.session == nil || h.cfg == nil || len(actors) == 0 {
		return nil, fmt.Errorf("council: invalid state")
	}
	snap := session.CopySettings(h.session.State.Settings)
	var combined strings.Builder
	var last *agent.RunResult

	for i, actorID := range actors {
		actorID = strings.TrimSpace(actorID)
		if actorID == "" {
			continue
		}
		doc, err := session.LoadChatTemplate(h.cfg.Root, actorID)
		if err != nil {
			return nil, err
		}
		label := strings.TrimSpace(doc.Name)
		if label == "" {
			label = actorID
		}
		if err := session.ApplyChatTemplate(h.session, h.cfg.Root, actorID); err != nil {
			return nil, err
		}
		h.emitActorSwitch(sendEvent, actorID)
		if i > 0 {
			sep := "\n\n"
			combined.WriteString(sep)
			if sendDelta != nil {
				_ = sendDelta(sep)
			}
		}
		header := fmt.Sprintf("**%s**\n\n", label)
		combined.WriteString(header)
		if sendDelta != nil {
			_ = sendDelta(header)
		}

		runGoal := goalText
		if i > 0 {
			runGoal = fmt.Sprintf(
				"[Council turn — %s] Respond in character to the discussion above. Do not repeat the user's question verbatim.",
				label,
			)
		}
		result, err := agent.Run(ctx, agent.RunRequest{
			Goal:             runGoal,
			ProviderID:       providerID,
			Session:          h.session,
			Providers:        rt.Providers,
			Tools:            rt.Tools,
			Skills:           rt.Skills,
			SkillsCatalog:    rt.SkillsCatalog,
			Config:           h.cfg,
			RAG:              rt.RAG,
			Memory:           rt.Memory,
			MCP:              rt.MCP,
			DecisionProfiles: rt.DecisionProfiles,
			PackIndex:        rt.PackIndex,
			Hooks:            rt.Hooks,
			Questions:        rt.Questions,
			WorkDir:          h.workDir,
			EventHandler:     sendEvent,
			StreamDelta:      sendDelta,
		})
		if err != nil {
			h.session.State.Settings = snap
			_ = h.session.Save()
			return nil, err
		}
		last = result
		if result != nil && strings.TrimSpace(result.Answer) != "" {
			combined.WriteString(strings.TrimSpace(result.Answer))
		}
	}

	h.session.State.Settings = snap
	if err := h.session.Save(); err != nil {
		return last, err
	}
	if last == nil {
		last = &agent.RunResult{Answer: combined.String()}
	} else {
		last.Answer = combined.String()
	}
	if h.session != nil {
		h.mu.Lock()
		h.activeSkills = skills.ActiveFromMessages(h.session.State.Messages)
		h.mu.Unlock()
	}
	return last, nil
}
