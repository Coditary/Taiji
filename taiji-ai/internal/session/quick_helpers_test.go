package session

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func TestEstimateTokensAndMessages(t *testing.T) {
	if EstimateTokens("") != 0 || EstimateTokens("abcd") != 1 {
		t.Fatal("estimate tokens")
	}
	msgs := []provider.Message{{Role: provider.RoleUser, Content: "hello world", Name: "u"}}
	if EstimateMessagesTokens(msgs) <= 0 {
		t.Fatal("messages tokens")
	}
	if contextWindowForModel("claude-3") != 200_000 {
		t.Fatal("claude window")
	}
	if contextWindowForModel("gemini-pro") != 1_000_000 {
		t.Fatal("gemini window")
	}
}

func TestRunContextRoundTrip(t *testing.T) {
	rt := AgentRuntime{BuiltinToolsAllow: []string{"grep"}}
	ctx := WithRunContext(context.Background(), rt)
	got, ok := RunContextFrom(ctx)
	if !ok || len(got.BuiltinToolsAllow) != 1 {
		t.Fatalf("got=%+v ok=%v", got, ok)
	}
	if _, ok := RunContextFrom(nil); ok {
		t.Fatal("nil ctx")
	}
}

func TestCompactHelpersAndTranscript(t *testing.T) {
	opts := DefaultCompactOptions()
	if opts.KeepRecentTurns != DefaultKeepRecentTurns {
		t.Fatal("defaults")
	}
	msg := FormatCompactUserMessage("summary bit")
	if !IsCompactUserMessage(msg) {
		t.Fatal("marker")
	}
	body, ok := ParseCompactUserMessage(msg)
	if !ok || body != "summary bit" {
		t.Fatalf("parse=%q ok=%v", body, ok)
	}
	tr := FormatTurnsTranscript([]ChatTurn{
		{User: "hi", Assistant: "hello"},
		{UserKind: "compact", User: "old"},
	})
	if tr == "" || !strings.Contains(tr, "User:") || !strings.Contains(tr, "Prior summary") {
		t.Fatalf("transcript=%q", tr)
	}
}

func testDummyRegistry(t *testing.T) *provider.Registry {
	t.Helper()
	cfg := &taijicfg.Config{
		Root:            t.TempDir(),
		DefaultProvider: "dummy_a",
		Providers: map[string]taijicfg.ProviderConfig{
			"dummy_a": {Type: "dummy", Model: "alpha"},
		},
	}
	reg, err := provider.NewRegistry(context.Background(), cfg, true)
	if err != nil {
		t.Fatal(err)
	}
	return reg
}

func TestGenerateCompactSummaryDummy(t *testing.T) {
	reg := testDummyRegistry(t)
	summary, err := GenerateCompactSummary(context.Background(), reg, "dummy_a", "User: hi\nAssistant: hello")
	if err != nil || summary == "" {
		t.Fatalf("err=%v summary=%q", err, summary)
	}
}

func TestCompactSessionRequiresHistory(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	reg := testDummyRegistry(t)
	if _, err := CompactSession(context.Background(), s, reg, "dummy_a", DefaultCompactOptions()); err == nil {
		t.Fatal("expected not enough history")
	}
}

func TestCompactSessionSummarizesOldTurns(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	s.State.Messages = ChatTurnsToMessages([]ChatTurn{
		{User: "one", Assistant: "a1"},
		{User: "two", Assistant: "a2"},
		{User: "three", Assistant: "a3"},
	})
	reg := testDummyRegistry(t)
	summary, err := CompactSession(context.Background(), s, reg, "dummy_a", CompactOptions{KeepRecentTurns: 1})
	if err != nil || summary == "" {
		t.Fatalf("err=%v summary=%q", err, summary)
	}
	if len(s.State.Messages) < 2 {
		t.Fatalf("messages=%d", len(s.State.Messages))
	}
}

func TestParseLoreEntryListOrMap(t *testing.T) {
	listRaw := json.RawMessage(`[{"content":"a","enabled":true},{"content":"b"}]`)
	if len(parseLoreEntryListOrMap(listRaw)) != 2 {
		t.Fatal("list")
	}
	mapRaw := json.RawMessage(`{"one":{"content":"x"},"two":{"content":"y","enabled":false}}`)
	got := parseLoreEntryListOrMap(mapRaw)
	if len(got) != 1 {
		t.Fatalf("map lore=%d", len(got))
	}
}

func TestClearDirBinding(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, "wd")
	s, err := New(root, "s1", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := SaveDirBinding(root, work, s.State.ID); err != nil {
		t.Fatal(err)
	}
	if err := ClearDirBinding(root, work); err != nil {
		t.Fatal(err)
	}
}
