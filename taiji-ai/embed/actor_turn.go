package main

import (
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/session"
)

func (h *runtimeHandle) emitActorSwitch(sendEvent func(events.Event), actorID string) {
	if sendEvent == nil || h == nil || h.session == nil {
		return
	}
	st := h.session.State.Settings
	sendEvent(events.Event{
		Type: events.TypeActorSwitch,
		Text: strings.TrimSpace(actorID),
		Data: map[string]any{
			"template_id":           strings.TrimSpace(st.TemplateID),
			"template_display_name": strings.TrimSpace(st.TemplateDisplayName),
			"avatar":                strings.TrimSpace(st.Avatar),
			"theme_color":           strings.TrimSpace(st.ThemeColor),
		},
	})
}

// prepareActorTurn applies a mentioned @actor template for one turn; restore merges actor state.
func (h *runtimeHandle) prepareActorTurn(projectRoot, goal string, sendEvent func(events.Event)) (string, func(), error) {
	if h == nil || h.session == nil {
		return goal, func() {}, nil
	}
	actorID, stripped := session.ResolveActorMention(projectRoot, goal)
	if actorID == "" {
		return goal, func() {}, nil
	}
	// Council path handles its own actor list; single @mention still routes one actor.
	if len(session.CouncilActorsForGoal(h.session, projectRoot, goal)) > 1 {
		return stripped, func() {}, nil
	}
	snap := session.CopySettings(h.session.State.Settings)
	if err := session.ApplyChatTemplate(h.session, projectRoot, actorID); err != nil {
		return "", nil, err
	}
	h.emitActorSwitch(sendEvent, actorID)
	restore := func() {
		overlayState := h.session.State.Settings.ActorState
		h.session.State.Settings = snap
		h.session.State.Settings.ActorState = session.MergeActorState(snap.ActorState, overlayState)
		_ = h.session.Save()
	}
	return stripped, restore, nil
}
