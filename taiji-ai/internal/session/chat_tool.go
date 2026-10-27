package session

import (
	"fmt"
	"regexp"
	"strings"
)

// ChatTool is a session-scoped tool defined at runtime (via create_chat_tool).
type ChatTool struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
	Language    string `json:"language" yaml:"language"` // python | shell
	Script      string `json:"script" yaml:"script"`
}

var chatToolNameRE = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

// ValidateChatToolName reports whether name is safe to expose to the model.
func ValidateChatToolName(name string) error {
	name = strings.TrimSpace(name)
	if !chatToolNameRE.MatchString(name) {
		return fmt.Errorf("tool name must match %s", chatToolNameRE.String())
	}
	return nil
}

// AddChatTool registers or replaces a session chat tool.
func (s *Session) AddChatTool(tool ChatTool) error {
	if s == nil {
		return fmt.Errorf("session is nil")
	}
	tool.Name = strings.TrimSpace(tool.Name)
	if err := ValidateChatToolName(tool.Name); err != nil {
		return err
	}
	tool.Language = strings.ToLower(strings.TrimSpace(tool.Language))
	if tool.Language != "python" && tool.Language != "shell" {
		return fmt.Errorf("language must be python or shell")
	}
	tool.Script = strings.TrimSpace(tool.Script)
	if tool.Script == "" {
		return fmt.Errorf("script is required")
	}
	if tool.Description == "" {
		tool.Description = "Session-defined chat tool"
	}
	for i, existing := range s.State.Settings.ChatTools {
		if strings.EqualFold(existing.Name, tool.Name) {
			s.State.Settings.ChatTools[i] = tool
			return s.Save()
		}
	}
	s.State.Settings.ChatTools = append(s.State.Settings.ChatTools, tool)
	return s.Save()
}

// ChatToolNames returns registered dynamic tool names for this session.
func (s *Session) ChatToolNames() []string {
	if s == nil {
		return nil
	}
	var names []string
	for _, t := range s.State.Settings.ChatTools {
		if n := strings.TrimSpace(t.Name); n != "" {
			names = append(names, n)
		}
	}
	return names
}
