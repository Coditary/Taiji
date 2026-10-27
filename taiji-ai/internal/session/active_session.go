package session

import "context"

type activeSessionKey struct{}

// WithActiveSession attaches the mutable session for builtin tools during a run.
func WithActiveSession(ctx context.Context, s *Session) context.Context {
	if ctx == nil || s == nil {
		return ctx
	}
	return context.WithValue(ctx, activeSessionKey{}, s)
}

// ActiveSessionFrom returns the session bound to the current agent run.
func ActiveSessionFrom(ctx context.Context) (*Session, bool) {
	if ctx == nil {
		return nil, false
	}
	s, ok := ctx.Value(activeSessionKey{}).(*Session)
	return s, ok && s != nil
}
