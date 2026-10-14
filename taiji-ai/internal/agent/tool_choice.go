package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/session"
)

const toolUseNudge = session.ToolUseNudgePrompt

// goalExpectsDecisions reports whether the user request likely needs the decide tool.
func goalExpectsDecisions(goal string) bool {
	g := strings.ToLower(strings.TrimSpace(goal))
	if g == "" {
		return false
	}
	excludes := []string{
		"was ist", "what is", "explain", "erklär", "how does", "wie funktioniert",
		"list ", "liste", "read ", "lies ", "grep", "glob", "show files", "run ",
		"shell", "command", "befehl", "datei", "file ", "directory", "verzeichnis",
		"hello", "hallo", "thanks", "danke", "2+2", "2 + 2",
		"was kostet", "what does", "how much", "welche datei", "which file",
		"contains", "enthält", "config file", "config.yaml",
	}
	for _, ex := range excludes {
		if strings.Contains(g, ex) {
			return false
		}
	}
	keywords := []string{
		"soll ich", "lohnt sich", "kaufen", "kauf ", "kaufentscheid", "ausgeben",
		"investieren", "budget", "zu teuer", "warten oder", "jetzt kaufen",
		"entscheid", "welche option", "was würdest du", "rate mir", "berate mich",
		"should i buy", "worth buying", "worth it", "purchase decision",
		"buy or wait", "buy or skip", "decide whether", "help me decide",
		"invest in", "afford", "can i afford", "should i get", "should i purchase",
		"which team", "welches team", "handle this", "route to", "wer soll",
		"payout", "failed", "fehlgeschlagen", "urgent", "dringend",
		"should we deploy", "deploy to prod", "hire", "einstellen", "accept the offer",
		"risk", "risiko", "priorität", "priority",
	}
	for _, kw := range keywords {
		if strings.Contains(g, kw) {
			return true
		}
	}
	return false
}

// goalExpectsTools reports whether the user request likely needs a filesystem or shell tool.
func goalExpectsTools(goal string) bool {
	g := strings.ToLower(strings.TrimSpace(goal))
	if g == "" {
		return false
	}
	keywords := []string{
		"list", "liste", "zeig", "show", "find", "search", "grep", "glob",
		"read", "lies", "open file", "datei", "directory", "verzeichnis",
		"ordner", "files", "dateien", "inhalt", "content of", "run ",
		"execute", "ausfüh", "shell", "command", "befehl", "cat ",
		"write file", "schreib", "projektstruktur", "project structure",
	}
	for _, kw := range keywords {
		if strings.Contains(g, kw) {
			return true
		}
	}
	return false
}

// respDescribesToolUse detects prose about tools instead of structured tool calls.
func respDescribesDecideUse(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	markers := []string{
		"`decide`", "decide tool", "decision engine", "decide({",
		"decide ({", "call decide", "rufe decide", "decide-Tool",
	}
	for _, m := range markers {
		if strings.Contains(c, m) {
			return true
		}
	}
	return false
}

func respDescribesToolUse(content string) bool {
	c := strings.ToLower(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	markers := []string{
		"i will use", "i would use", "i'll use", "let me use",
		"ich werde", "ich würde", "ich kann als", "cannot execute",
		"kann nicht ausführen", "kann keine befehle", "tool to ",
		"`list_dir`", "`read_file`", "`grep`", "`glob_search`", "`shell`",
		"[tool call]",
	}
	for _, m := range markers {
		if strings.Contains(c, m) {
			return true
		}
	}
	return false
}

func resolveToolChoice(goal string, tools []provider.ToolDef, messages []provider.Message, step int, nudged bool) string {
	if len(tools) == 0 {
		return ""
	}
	if nudged {
		return "required"
	}
	if step == 1 && (goalExpectsTools(goal) || goalExpectsDecisions(goal)) {
		return "required"
	}
	return "auto"
}

func hasRecentToolObservation(messages []provider.Message) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		switch messages[i].Role {
		case provider.RoleTool:
			return true
		case provider.RoleAssistant:
			if len(messages[i].ToolCalls) > 0 {
				continue
			}
			return false
		case provider.RoleUser:
			return false
		}
	}
	return false
}

func lastToolObservation(messages []provider.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == provider.RoleTool {
			return strings.TrimSpace(messages[i].Content)
		}
	}
	return ""
}

func duplicateToolObservation(messages []provider.Message, call provider.ToolCall) string {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role != provider.RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		for _, prev := range m.ToolCalls {
			if prev.Name != call.Name || strings.TrimSpace(prev.Args) != strings.TrimSpace(call.Args) {
				continue
			}
			for j := i + 1; j < len(messages); j++ {
				if messages[j].Role == provider.RoleTool && messages[j].ToolCallID == prev.ID {
					return messages[j].Content
				}
			}
		}
	}
	return "duplicate tool call skipped"
}

func isDuplicateToolCall(messages []provider.Message, call provider.ToolCall) bool {
	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Role != provider.RoleAssistant || len(m.ToolCalls) == 0 {
			continue
		}
		for _, prev := range m.ToolCalls {
			if prev.Name != call.Name || strings.TrimSpace(prev.Args) != strings.TrimSpace(call.Args) {
				continue
			}
			for j := i + 1; j < len(messages); j++ {
				if messages[j].Role == provider.RoleTool && messages[j].ToolCallID == prev.ID {
					return true
				}
			}
		}
	}
	return false
}

func buildCompletionRequest(
	req RunRequest,
	p provider.Provider,
	messages []provider.Message,
	toolDefs []provider.ToolDef,
	step int,
	toolNudged bool,
	onDelta func(string) error,
	onThinkDelta func(string) error,
) provider.CompletionRequest {
	contextWindow := 0
	if req.Config != nil {
		contextWindow = session.ResolveContextWindow(req.Config, p.ID())
	}
	toolChoice := resolveToolChoice(req.Goal, toolDefs, messages, step, toolNudged)
	tools := toolDefs
	if toolChoice == "none" {
		tools = nil
		toolChoice = ""
	}
	return provider.CompletionRequest{
		Messages:      messages,
		Tools:         tools,
		ContextWindow: contextWindow,
		ToolChoice:    toolChoice,
		OnDelta:       onDelta,
		OnThinkDelta:  onThinkDelta,
	}
}
