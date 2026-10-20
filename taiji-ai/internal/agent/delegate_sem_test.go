package agent

import (
	"strings"
	"testing"
	"time"
)

func TestDelegateLimiterBlocksUntilRelease(t *testing.T) {
	lim := newDelegateLimiter(1)
	r1 := lim.acquire("sess")
	done := make(chan struct{})
	go func() {
		r2 := lim.acquire("sess")
		r2()
		close(done)
	}()
	select {
	case <-done:
		t.Fatal("second acquire should block")
	case <-time.After(50 * time.Millisecond):
	}
	r1()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("second acquire did not proceed")
	}
}

func TestSubSessionIDFormat(t *testing.T) {
	lim := newDelegateLimiter(3)
	id1 := lim.nextSubSessionID("parent")
	id2 := lim.nextSubSessionID("parent")
	if id1 == id2 || !strings.Contains(id1, "parent:sub:") {
		t.Fatalf("ids: %s %s", id1, id2)
	}
}
