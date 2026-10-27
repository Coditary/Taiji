package provider

import "testing"

func TestChatformatBridgeRoundTrip(t *testing.T) {
	orig := []Message{{
		Role: RoleUser,
		Content: "hi",
		ToolCalls: []ToolCall{{ID: "1", Name: "read_file", Args: `{}`}},
	}}
	cf := toChatformatMessages(orig)
	back := fromChatformatMessages(cf)
	if len(back) != 1 || back[0].Content != "hi" || back[0].ToolCalls[0].Name != "read_file" {
		t.Fatalf("back=%+v", back)
	}
	tools := toChatformatTools([]ToolDef{{Name: "glob", Description: "find files"}})
	if len(tools) != 1 || tools[0].Name != "glob" {
		t.Fatalf("tools=%+v", tools)
	}
}
