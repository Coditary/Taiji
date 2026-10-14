package question

import "context"

type ctxKey int

const (
	sessionIDKey ctxKey = iota
	onAskKey
	toolCallIDKey
)

// RunConfig carries per-run question wiring through context.
type RunConfig struct {
	SessionID string
	OnAsk     func(Request)
}

// AttachRun stores question run configuration on ctx.
func AttachRun(ctx context.Context, cfg RunConfig) context.Context {
	if cfg.SessionID != "" {
		ctx = context.WithValue(ctx, sessionIDKey, cfg.SessionID)
	}
	if cfg.OnAsk != nil {
		ctx = context.WithValue(ctx, onAskKey, cfg.OnAsk)
	}
	return ctx
}

// WithToolCall stores the active tool call id on ctx.
func WithToolCall(ctx context.Context, callID string) context.Context {
	if callID == "" {
		return ctx
	}
	return context.WithValue(ctx, toolCallIDKey, callID)
}

// SessionIDFrom returns the session id from ctx, if any.
func SessionIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(sessionIDKey).(string); ok {
		return v
	}
	return ""
}

// ToolCallIDFrom returns the active tool call id from ctx, if any.
func ToolCallIDFrom(ctx context.Context) string {
	if v, ok := ctx.Value(toolCallIDKey).(string); ok {
		return v
	}
	return ""
}

func onAskFrom(ctx context.Context) func(Request) {
	if fn, ok := ctx.Value(onAskKey).(func(Request)); ok {
		return fn
	}
	return nil
}
