package agent

import (
	"context"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

func TestKeepRecentTurnsFromMessages(t *testing.T) {
	if keepRecentTurnsFromMessages(0) != 3 {
		t.Fatalf("default keep turns")
	}
	if keepRecentTurnsFromMessages(1) != 1 {
		t.Fatal("single message")
	}
	if keepRecentTurnsFromMessages(4) != 2 {
		t.Fatalf("four messages -> two turns")
	}
}

func TestCompactSessionHistoryErrors(t *testing.T) {
	if _, err := CompactSessionHistory(context.Background(), nil, nil, "", nil); err == nil {
		t.Fatal("nil session")
	}
	root := t.TempDir()
	s, err := session.New(root, "s1", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	reg := testDummyRegistry(t)
	if _, err := CompactSessionHistory(context.Background(), s, reg, "dummy_a", &taijicfg.Config{}); err == nil {
		t.Fatal("empty history")
	}
}

func TestCompactSessionHistoryOfflineSummary(t *testing.T) {
	root := t.TempDir()
	s, err := session.New(root, "s1", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	var msgs []provider.Message
	for i := 0; i < 8; i++ {
		msgs = append(msgs,
			provider.Message{Role: provider.RoleUser, Content: "user turn"},
			provider.Message{Role: provider.RoleAssistant, Content: "assistant reply"},
		)
	}
	s.State.Messages = msgs
	reg := testDummyRegistry(t)
	summary, err := CompactSessionHistory(context.Background(), s, reg, "dummy_a", &taijicfg.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "Offline compaction") && !strings.Contains(summary, "user turn") {
		t.Fatalf("summary=%q", summary)
	}
	if len(s.State.Messages) < 2 {
		t.Fatal("expected compacted messages retained")
	}
}
