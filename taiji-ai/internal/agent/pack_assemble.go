package agent

import (
	"context"
	"strings"

	"github.com/coditary/taiji/internal/events"
	"github.com/coditary/taiji/internal/packs"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

type packAssembleResult struct {
	systemAppendix string
	userPreface    string
	summary        string
}

func runPackAssembly(
	ctx context.Context,
	req RunRequest,
	goal string,
	emit func(events.Event),
) packAssembleResult {
	out := packAssembleResult{}
	if req.PackIndex == nil || !req.PackIndex.Enabled() {
		return out
	}
	_ = req.PackIndex.ReloadAt(projectWorkspaceRoot(req))
	allow := packAllowList(req)
	recent := recentUserText(req.Session, 3)
	assembled, err := req.PackIndex.Assemble(ctx, packs.AssembleRequest{
		Goal:          goal,
		RecentText:    recent,
		PacksAllow:    allow,
		SkillLoader:   req.Skills,
		ResolveSkills: req.SkillNames,
	})
	if err != nil {
		if emit != nil {
			emit(events.Event{Type: events.TypeError, Text: "pack assembly: " + err.Error()})
		}
		return out
	}
	out.systemAppendix = assembled.SystemAppendix
	out.userPreface = assembled.UserPreface
	out.summary = packs.FormatInjectSummary(assembled.Lines)
	if out.summary != "" && emit != nil {
		emit(events.Event{
			Type: events.TypeObservation,
			Text: out.summary,
			Data: map[string]any{"pack_inject": assembled.Lines},
		})
	}
	return out
}

func packAllowList(req RunRequest) []string {
	if req.Session == nil {
		return nil
	}
	rt := req.Session.AgentRuntimeForRun(req.Config)
	return rt.PacksAllow
}

func recentUserText(sess *session.Session, maxTurns int) string {
	if sess == nil || len(sess.State.Messages) == 0 {
		return ""
	}
	var parts []string
	for i := len(sess.State.Messages) - 1; i >= 0 && len(parts) < maxTurns; i-- {
		m := sess.State.Messages[i]
		if m.Role != provider.RoleUser {
			continue
		}
		if text := strings.TrimSpace(m.Content); text != "" {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}
