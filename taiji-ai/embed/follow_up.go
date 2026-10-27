package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/agent"
	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/runtime"
	"github.com/coditary/taiji/internal/session"
)

func (h *runtimeHandle) runFollowUpActor(
	ctx context.Context,
	providerID string,
	rt *runtime.Runtime,
	priorAnswer string,
	sendDelta func(string) error,
	sendEvent func(events.Event),
) (*agent.RunResult, error) {
	if h == nil || h.session == nil || h.cfg == nil || rt == nil {
		return nil, nil
	}
	followID := strings.TrimSpace(h.session.State.Settings.FollowUpActor)
	if followID == "" {
		return nil, nil
	}
	if _, err := session.LoadChatTemplate(h.cfg.Root, followID); err != nil {
		return nil, nil
	}
	snap := session.CopySettings(h.session.State.Settings)
	if err := session.ApplyChatTemplate(h.session, h.cfg.Root, followID); err != nil {
		return nil, err
	}
	h.emitActorSwitch(sendEvent, followID)
	if sendDelta != nil {
		_ = sendDelta("\n\n---\n\n")
	}
	goal := fmt.Sprintf(
		"[Follow-up actor %s] Review the assistant's previous reply and respond in character. Prior reply:\n\n%s",
		followID,
		strings.TrimSpace(priorAnswer),
	)
	result, err := agent.Run(ctx, agent.RunRequest{
		Goal:             goal,
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
	overlayState := h.session.State.Settings.ActorState
	h.session.State.Settings = snap
	h.session.State.Settings.ActorState = session.MergeActorState(snap.ActorState, overlayState)
	_ = h.session.Save()
	return result, err
}
