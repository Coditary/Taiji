package session

import "context"

type runContextKey struct{}

// WithRunContext attaches per-run agent policy (tool allowlists) to ctx.
func WithRunContext(ctx context.Context, rt AgentRuntime) context.Context {
	return context.WithValue(ctx, runContextKey{}, rt)
}

// RunContextFrom returns the agent runtime policy for the current tool call.
func RunContextFrom(ctx context.Context) (AgentRuntime, bool) {
	if ctx == nil {
		return AgentRuntime{}, false
	}
	rt, ok := ctx.Value(runContextKey{}).(AgentRuntime)
	return rt, ok
}
