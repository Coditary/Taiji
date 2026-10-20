package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/coditary/taiji/internal/tools/builtin"
)

const defaultDelegateMaxConcurrent = 3

var delegateSlots = newDelegateLimiter(defaultDelegateMaxConcurrent)

type delegateLimiter struct {
	mu      sync.Mutex
	max     int
	bySess  map[string]int
	subSeq  map[string]uint64
	waiting map[string]chan struct{}
}

func newDelegateLimiter(max int) *delegateLimiter {
	if max <= 0 {
		max = defaultDelegateMaxConcurrent
	}
	return &delegateLimiter{
		max:     max,
		bySess:  map[string]int{},
		subSeq:  map[string]uint64{},
		waiting: map[string]chan struct{}{},
	}
}

func (l *delegateLimiter) acquire(sessionID string) (release func()) {
	sessionID = normalizeDelegateSession(sessionID)
	l.mu.Lock()
	for l.bySess[sessionID] >= l.max {
		ch := l.waiting[sessionID]
		if ch == nil {
			ch = make(chan struct{})
			l.waiting[sessionID] = ch
		}
		l.mu.Unlock()
		<-ch
		l.mu.Lock()
	}
	l.bySess[sessionID]++
	l.mu.Unlock()
	return func() {
		l.mu.Lock()
		l.bySess[sessionID]--
		if l.bySess[sessionID] < 0 {
			l.bySess[sessionID] = 0
		}
		if ch := l.waiting[sessionID]; ch != nil {
			close(ch)
			delete(l.waiting, sessionID)
		}
		l.mu.Unlock()
	}
}

func (l *delegateLimiter) nextSubSessionID(parent string) string {
	parent = normalizeDelegateSession(parent)
	l.mu.Lock()
	n := l.subSeq[parent]
	n++
	l.subSeq[parent] = n
	l.mu.Unlock()
	return fmt.Sprintf("%s:sub:%d", parent, n)
}

func normalizeDelegateSession(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "default"
	}
	return id
}

// SetDelegateMaxConcurrent configures the process-wide delegate_task limiter (tests).
func SetDelegateMaxConcurrent(max int) {
	delegateSlots = newDelegateLimiter(max)
}

func subSessionContext(ctx context.Context, parentSessionID string) (context.Context, string) {
	subID := delegateSlots.nextSubSessionID(parentSessionID)
	return builtin.WithToolSessionID(ctx, subID), subID
}
