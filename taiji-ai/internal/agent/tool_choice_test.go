package agent

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestGoalExpectsTools(t *testing.T) {
	cases := []struct {
		goal string
		want bool
	}{
		{"List files in the current directory", true},
		{"Zeig mir die Dateien hier", true},
		{"grep for func Register", true},
		{"Hello", false},
		{"What is 2+2?", false},
	}
	for _, tc := range cases {
		if got := goalExpectsTools(tc.goal); got != tc.want {
			t.Fatalf("goalExpectsTools(%q) = %v, want %v", tc.goal, got, tc.want)
		}
	}
}

func TestRespDescribesToolUse(t *testing.T) {
	if !respDescribesToolUse("I will use the `list_dir` tool to list files.") {
		t.Fatal("expected prose tool description to be detected")
	}
	if respDescribesToolUse("Here are the files: a.go, b.go") {
		t.Fatal("plain answer should not be flagged")
	}
}

func TestIsDuplicateToolCall(t *testing.T) {
	call := provider.ToolCall{ID: "1", Name: "list_dir", Args: `{"path":"."}`}
	msgs := []provider.Message{
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{call}},
	}
	if isDuplicateToolCall(msgs, call) {
		t.Fatal("fresh tool call should not be duplicate before observation")
	}
	msgs = append(msgs, provider.Message{Role: provider.RoleTool, Name: "list_dir", ToolCallID: "1", Content: "a.go"})
	if !isDuplicateToolCall(msgs, call) {
		t.Fatal("repeated tool call after observation should be duplicate")
	}
}

func TestHasRecentToolObservation(t *testing.T) {
	call := provider.ToolCall{ID: "1", Name: "list_dir", Args: `{"path":"."}`}
	msgs := []provider.Message{
		{Role: provider.RoleUser, Content: "list files"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{call}},
		{Role: provider.RoleTool, Name: "list_dir", ToolCallID: "1", Content: "a.go"},
		{Role: provider.RoleAssistant, ToolCalls: []provider.ToolCall{{ID: "2", Name: "list_dir", Args: `{"path":"."}`}}},
	}
	if !hasRecentToolObservation(msgs) {
		t.Fatal("should find tool observation below pending assistant tool calls")
	}
}

func TestResolveToolChoice(t *testing.T) {
	tools := []provider.ToolDef{{Name: "list_dir"}}
	if resolveToolChoice("list files", tools, nil, 1, false) != "required" {
		t.Fatal("step 1 file goal should require tools")
	}
	if resolveToolChoice("hello", tools, nil, 1, false) != "auto" {
		t.Fatal("greeting should use auto tool choice")
	}
	if resolveToolChoice("hello", tools, nil, 1, true) != "required" {
		t.Fatal("nudged retry should require tools")
	}
	msgs := []provider.Message{{Role: provider.RoleTool, Name: "list_dir", Content: "a.go"}}
	if resolveToolChoice("list files", tools, msgs, 2, false) != "none" {
		t.Fatal("after tool observation tool choice should be none")
	}
}
