package agent

import (
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestPruneStaleReadFileAfterEdit(t *testing.T) {
	messages := []provider.Message{
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "read_file", Args: `{"path":"a.go"}`}}},
		{Role: provider.RoleTool, ToolCallID: "1", Name: "read_file", Content: strings.Repeat("x", 100)},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "2", Name: "edit_file", Args: `{"path":"a.go"}`}}},
		{Role: provider.RoleTool, ToolCallID: "2", Name: "edit_file", Content: "edited a.go"},
	}
	out := pruneStaleToolObservations(messages, 2, pruneOptions{})
	if !strings.Contains(out[1].Content, "Pruned stale read_file") {
		t.Fatalf("expected pruned read, got %q", out[1].Content)
	}
}

func TestPruneOversizedOldShellOutput(t *testing.T) {
	big := strings.Repeat("log line\n", 500)
	messages := []provider.Message{
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "shell", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "1", Name: "shell", Content: big},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "2", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "2", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "3", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "3", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "4", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "4", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "5", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "5", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "6", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "6", Name: "list_dir", Content: "ok"},
	}
	out := pruneStaleToolObservations(messages, 6, pruneOptions{})
	if !strings.Contains(out[1].Content, "lines pruned from older step") {
		t.Fatalf("expected pruned shell output, len=%d", len(out[1].Content))
	}
}

func TestCacheFriendlyPruningSkipsOffInterval(t *testing.T) {
	big := strings.Repeat("log line\n", 500)
	messages := []provider.Message{
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "1", Name: "shell", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "1", Name: "shell", Content: big},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "2", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "2", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "3", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "3", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "4", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "4", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "5", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "5", Name: "list_dir", Content: "ok"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "6", Name: "list_dir", Args: `{}`}}},
		{Role: provider.RoleTool, ToolCallID: "6", Name: "list_dir", Content: "ok"},
	}
	out := pruneStaleToolObservations(messages, 5, pruneOptions{cacheFriendly: true})
	if strings.Contains(out[1].Content, pruneSizeMarker) {
		t.Fatal("cache-friendly pruning should skip size prune off 4-step boundary")
	}
	out = pruneStaleToolObservations(messages, 8, pruneOptions{cacheFriendly: true})
	if !strings.Contains(out[1].Content, pruneSizeMarker) {
		t.Fatal("expected size prune on step 8 boundary")
	}
	first := out[1].Content
	out = pruneStaleToolObservations(messages, 12, pruneOptions{cacheFriendly: true})
	if out[1].Content != first {
		t.Fatal("already-pruned content should remain stable across later prune passes")
	}
}
