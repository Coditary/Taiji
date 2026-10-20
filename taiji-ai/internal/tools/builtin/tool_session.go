package builtin

import "context"

type toolSessionKey struct{}

// WithToolSessionID attaches the chat session id for background jobs and similar tools.
func WithToolSessionID(ctx context.Context, sessionID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, toolSessionKey{}, sessionID)
}

// ToolSessionIDFrom returns the session id for the current tool call.
func ToolSessionIDFrom(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	id, _ := ctx.Value(toolSessionKey{}).(string)
	return id
}
