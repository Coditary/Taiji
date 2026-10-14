package hooks

import "encoding/json"

// Event names for declarative and in-process hooks.
const (
	EventPromptBefore  = "prompt.before"
	EventContextBefore = "context.before"
	EventToolBefore    = "tool.before"
	EventToolAfter     = "tool.after"
	EventFileChanged   = "file.changed"
	EventSessionStart  = "session.start"
	EventSessionIdle   = "session.idle"
	EventSessionError  = "session.error"
)

// Env carries run metadata available to hook handlers.
type Env struct {
	SessionID string
	WorkDir   string
	Goal      string
	Step      int
}

// PromptData is the mutable payload for prompt.before.
type PromptData struct {
	Env
	Text string
}

// ContextData is the payload for context.before.
type ContextData struct {
	Env
	// AdditionalSystem is appended to the system prompt for the next model call.
	AdditionalSystem string
}

// ToolData is the mutable payload for tool.before and tool.after.
type ToolData struct {
	Env
	Tool   string
	ToolID string
	Args   string
	Output string
}

// FileChangedData is emitted after successful write/edit tools.
type FileChangedData struct {
	Env
	Tool  string
	Files []string
}

// SessionData is the payload for session.* lifecycle hooks.
type SessionData struct {
	Env
	Answer string
	Error  string
}

// ShellInput is JSON written to shell hook stdin.
type ShellInput struct {
	Event string         `json:"event"`
	Env   Env            `json:"env"`
	Tool  *ToolData      `json:"tool,omitempty"`
	Prompt *PromptData   `json:"prompt,omitempty"`
	Context *ContextData `json:"context,omitempty"`
	File  *FileChangedData `json:"file,omitempty"`
	Session *SessionData `json:"session,omitempty"`
}

// Result is returned by hook handlers (shell JSON or in-process plugins).
type Result struct {
	Block             bool   `json:"block,omitempty"`
	Message           string `json:"message,omitempty"`
	UpdatedPrompt     string `json:"updated_prompt,omitempty"`
	AdditionalSystem  string `json:"additional_system,omitempty"`
	UpdatedArgs       string `json:"updated_args,omitempty"`
	UpdatedOutput     string `json:"updated_output,omitempty"`
	AdditionalContext string `json:"additional_context,omitempty"`
}

func (r Result) mergeInto(other Result) Result {
	out := other
	if r.Block {
		out.Block = true
	}
	if r.Message != "" {
		out.Message = r.Message
	}
	if r.UpdatedPrompt != "" {
		out.UpdatedPrompt = r.UpdatedPrompt
	}
	if r.AdditionalSystem != "" {
		out.AdditionalSystem = r.AdditionalSystem
	}
	if r.UpdatedArgs != "" {
		out.UpdatedArgs = r.UpdatedArgs
	}
	if r.UpdatedOutput != "" {
		out.UpdatedOutput = r.UpdatedOutput
	}
	if r.AdditionalContext != "" {
		if out.AdditionalContext != "" {
			out.AdditionalContext += "\n" + r.AdditionalContext
		} else {
			out.AdditionalContext = r.AdditionalContext
		}
	}
	return out
}

func decodeShellResult(stdout string) (Result, error) {
	trimmed := trimSpace(stdout)
	if trimmed == "" {
		return Result{}, nil
	}
	if trimmed[0] != '{' {
		return Result{AdditionalContext: trimmed}, nil
	}
	var out Result
	if err := json.Unmarshal([]byte(trimmed), &out); err != nil {
		return Result{}, err
	}
	return out, nil
}

func trimSpace(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\r' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\r' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
