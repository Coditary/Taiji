package session

import (
	"strings"
	"testing"
)

func TestValidateChatToolName(t *testing.T) {
	if err := ValidateChatToolName("helper_1"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateChatToolName("Bad"); err == nil {
		t.Fatal("expected invalid name")
	}
}

func TestAddChatToolAndNames(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "", "dummy", "chat")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.AddChatTool(ChatTool{Name: "run_check", Language: "shell", Script: "echo ok"}); err != nil {
		t.Fatal(err)
	}
	names := s.ChatToolNames()
	if len(names) != 1 || names[0] != "run_check" {
		t.Fatalf("names=%v", names)
	}
	if err := s.AddChatTool(ChatTool{Name: "run_check", Language: "python", Script: "print(1)"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(s.State.Settings.ChatTools[0].Script, "print") {
		t.Fatal("replace existing")
	}
	if err := s.AddChatTool(ChatTool{Name: "run_check", Language: "ruby", Script: "x"}); err == nil {
		t.Fatal("bad language")
	}
}
