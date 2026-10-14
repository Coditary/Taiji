package question

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Prompter collects answers from the user for a request.
type Prompter interface {
	Prompt(ctx context.Context, req Request) ([]Answer, error)
}

type replyResult struct {
	answers []Answer
	err     error
}

type pendingEntry struct {
	req Request
	ch  chan replyResult
}

// Service coordinates user prompts during agent runs.
type Service struct {
	mu       sync.Mutex
	pending  map[string]*pendingEntry
	prompter Prompter
}

// NewService creates a question service.
func NewService(p Prompter) *Service {
	return &Service{
		pending:  map[string]*pendingEntry{},
		prompter: p,
	}
}

// SetPrompter replaces the active prompter.
func (s *Service) SetPrompter(p Prompter) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.prompter = p
	s.mu.Unlock()
}

// Prompter returns the configured prompter, if any.
func (s *Service) Prompter() Prompter {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.prompter
}

// Ask blocks until the user answers or rejects the prompt.
func (s *Service) Ask(ctx context.Context, input AskInput) ([]Answer, error) {
	if s == nil {
		return nil, ErrNoPrompter
	}
	if len(input.Questions) == 0 {
		return nil, ErrInvalidQuestions
	}

	req := Request{
		ID:        newRequestID(),
		SessionID: input.SessionID,
		Questions: append([]Info(nil), input.Questions...),
		Tool:      input.Tool,
	}
	if fn := onAskFrom(ctx); fn != nil {
		fn(req)
	}

	s.mu.Lock()
	prompter := s.prompter
	s.mu.Unlock()

	if prompter != nil {
		return prompter.Prompt(ctx, req)
	}

	ch := make(chan replyResult, 1)
	s.mu.Lock()
	s.pending[req.ID] = &pendingEntry{req: req, ch: ch}
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.pending, req.ID)
		s.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case res := <-ch:
		return res.answers, res.err
	}
}

// Reply completes a pending deferred request.
func (s *Service) Reply(input ReplyInput) error {
	if s == nil {
		return ErrNotFound
	}
	s.mu.Lock()
	entry, ok := s.pending[input.RequestID]
	if ok {
		delete(s.pending, input.RequestID)
	}
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	entry.ch <- replyResult{answers: cloneAnswers(input.Answers), err: nil}
	return nil
}

// Reject dismisses a pending deferred request.
func (s *Service) Reject(requestID string) error {
	if s == nil {
		return ErrNotFound
	}
	s.mu.Lock()
	entry, ok := s.pending[requestID]
	if ok {
		delete(s.pending, requestID)
	}
	s.mu.Unlock()
	if !ok {
		return ErrNotFound
	}
	entry.ch <- replyResult{err: ErrRejected}
	return nil
}

// ListPending returns outstanding deferred requests.
func (s *Service) ListPending() []Request {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Request, 0, len(s.pending))
	for _, entry := range s.pending {
		out = append(out, entry.req)
	}
	return out
}

func newRequestID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString(b[:])
	}
	return hex.EncodeToString(b[:])
}

func cloneAnswers(in []Answer) []Answer {
	out := make([]Answer, len(in))
	for i, ans := range in {
		out[i] = append([]string(nil), ans...)
	}
	return out
}
