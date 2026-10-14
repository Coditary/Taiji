package decision

import "context"

// Backend runs structured decision requests against a configured engine.
type Backend interface {
	Decide(ctx context.Context, req Request) (*Response, error)
}
