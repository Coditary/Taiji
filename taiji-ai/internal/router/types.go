package router

// Action names a turn-level orchestration step executed after routing.
type Action string

const (
	ActionMemorize        Action = "memorize"
	ActionRecall          Action = "recall"
	ActionToolCall        Action = "tool_call"
	ActionDecide          Action = "decide"
	ActionInvokeSkill     Action = "invoke_skill"
	ActionAskUser         Action = "ask_user"
	ActionRAGQuery        Action = "rag_query"
	ActionAnswerDirect    Action = "answer_direct"
	ActionContinueSession Action = "continue_session"
	ActionHook            Action = "hook"
	ActionUnknown         Action = "unknown"
	ActionStop            Action = "stop"
)

// SkillMatch is one ranked skill candidate from turn-start search.
type SkillMatch struct {
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`
	Score       float32 `json:"score"`
	MatchSource string  `json:"match_source,omitempty"`
	Fit         string  `json:"fit,omitempty"`
}

// Plan is the router decision for one user turn.
type Plan struct {
	Action        Action       `json:"action"`
	Reason        string       `json:"reason,omitempty"`
	MemoryKind    string       `json:"memory_kind,omitempty"`
	SkillName     string       `json:"skill_name,omitempty"`
	SkillMatches  []SkillMatch `json:"skill_matches,omitempty"`
	DecideProfile string       `json:"decide_profile,omitempty"`
	ToolHint      string       `json:"tool_hint,omitempty"`
	Message       string       `json:"message,omitempty"`
}
