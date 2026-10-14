package agent

import (
	"context"
	"strings"

	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/provider"
)

func hookEnvFor(req RunRequest, step int, goal string) hooks.Env {
	env := hooks.Env{
		WorkDir: strings.TrimSpace(req.WorkDir),
		Goal:    strings.TrimSpace(goal),
		Step:    step,
	}
	if req.Session != nil {
		env.SessionID = req.Session.State.ID
	}
	return env
}

func runSessionErrorHook(ctx context.Context, req RunRequest, env hooks.Env, err error) {
	if req.Hooks == nil || err == nil {
		return
	}
	_ = req.Hooks.RunSession(ctx, hooks.EventSessionError, hooks.SessionData{
		Env:   env,
		Error: err.Error(),
	})
}

func appendHookContext(messages []provider.Message, extra string) []provider.Message {
	extra = strings.TrimSpace(extra)
	if extra == "" {
		return messages
	}
	out := append([]provider.Message{}, messages...)
	for i, msg := range out {
		if msg.Role == provider.RoleSystem {
			out[i].Content = strings.TrimSpace(msg.Content + "\n\n" + extra)
			return out
		}
	}
	return append([]provider.Message{{Role: provider.RoleSystem, Content: extra}}, out...)
}
