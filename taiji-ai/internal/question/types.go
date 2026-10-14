package question

// Option is a selectable answer choice.
type Option struct {
	ID          string `json:"id,omitempty"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
}

// Info describes one question shown to the user.
type Info struct {
	ID       string   `json:"id,omitempty"`
	Header   string   `json:"header,omitempty"`
	Question string   `json:"question"`
	Options  []Option `json:"options"`
	Multiple bool     `json:"multiple,omitempty"`
	Custom   *bool    `json:"custom,omitempty"`
}

// AllowsCustom reports whether free-text answers are allowed (default true).
func (q Info) AllowsCustom() bool {
	if q.Custom == nil {
		return true
	}
	return *q.Custom
}

// ToolRef correlates a prompt with a tool call.
type ToolRef struct {
	CallID string `json:"call_id,omitempty"`
}

// Request is an outstanding user prompt.
type Request struct {
	ID        string   `json:"id"`
	SessionID string   `json:"session_id,omitempty"`
	Questions []Info   `json:"questions"`
	Tool      *ToolRef `json:"tool,omitempty"`
}

// Answer holds selected labels for one question.
type Answer = []string

// AskInput configures Ask.
type AskInput struct {
	SessionID string
	Questions []Info
	Tool      *ToolRef
}

// ReplyInput carries user answers back to a pending request.
type ReplyInput struct {
	RequestID string
	Answers   []Answer
}
