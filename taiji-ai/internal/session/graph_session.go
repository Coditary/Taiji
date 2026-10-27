package session

import (
	"strings"

	"github.com/coditary/taiji/internal/graph"
)

// GraphSession binds a blueprint id to mutable world state for this chat.
type GraphSession struct {
	BlueprintID   string            `json:"blueprint_id"`
	BlueprintPath string            `json:"blueprint_path,omitempty"`
	World         *graph.WorldState `json:"world"`
}

// GraphActive reports whether guided graph mode is enabled for the session.
func (s *Session) GraphActive() bool {
	if s == nil || s.State.Graph == nil {
		return false
	}
	return strings.TrimSpace(s.State.Graph.BlueprintID) != "" && s.State.Graph.World != nil
}

// GraphBuiltinToolNames are exposed while a graph is loaded.
func GraphBuiltinToolNames() []string {
	return []string{
		"graph_complete_node",
		"graph_move_entity",
		"graph_extend_blueprint",
	}
}

// EnsureGraphToolsAllowed adds graph tools to session builtin allow list when needed.
func (s *Session) EnsureGraphToolsAllowed() {
	if s == nil || !s.GraphActive() {
		return
	}
	names := GraphBuiltinToolNames()
	allow := s.State.Settings.BuiltinToolsAllow
	if len(allow) == 0 {
		return
	}
	for _, name := range names {
		found := false
		for _, a := range allow {
			if strings.EqualFold(a, name) {
				found = true
				break
			}
		}
		if !found {
			allow = append(allow, name)
		}
	}
	s.State.Settings.BuiltinToolsAllow = allow
}

// BuildGraphContextAppendix returns runtime scene assembly for the system prompt.
func (s *Session) BuildGraphContextAppendix(projectRoot string) string {
	if !s.GraphActive() {
		return ""
	}
	bp, _, err := graph.LoadBlueprintByID(projectRoot, s.State.Graph.BlueprintID)
	if err != nil {
		return ""
	}
	return graph.AssembleContext(bp, s.State.Graph.World)
}
