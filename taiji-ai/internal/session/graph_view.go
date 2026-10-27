package session

import (
	"encoding/json"
	"strings"

	"github.com/coditary/taiji/internal/graph"
)

// GraphViewNode is a neighbor in the radar widget.
type GraphViewNode struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Label   string `json:"label,omitempty"`
	Visited bool   `json:"visited,omitempty"`
	Current bool   `json:"current,omitempty"`
	Locked  bool   `json:"locked,omitempty"`
}

// GraphViewEntity is shown in the entity column.
type GraphViewEntity struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Kind        string `json:"kind,omitempty"`
	Persistence string `json:"persistence,omitempty"`
	Memory      string `json:"memory,omitempty"`
	Role        string `json:"role,omitempty"`
	TemplateID  string `json:"template_id,omitempty"`
}

// GraphView is the TUI payload for radar + entity panels.
type GraphView struct {
	Active         bool              `json:"active"`
	GraphID        string            `json:"graph_id,omitempty"`
	GraphName      string            `json:"graph_name,omitempty"`
	CurrentNodeID  string            `json:"current_node_id,omitempty"`
	CurrentTitle   string            `json:"current_title,omitempty"`
	NodeState      string            `json:"node_state,omitempty"`
	Predecessors   []GraphViewNode   `json:"predecessors,omitempty"`
	Successors     []GraphViewNode   `json:"successors,omitempty"`
	HereEntities   []GraphViewEntity `json:"here_entities,omitempty"`
	PartyEntities  []GraphViewEntity `json:"party_entities,omitempty"`
	NodeTools      []string          `json:"node_tools,omitempty"`
	ExitHint       string            `json:"exit_hint,omitempty"`
	EntityCount    int               `json:"entity_count"`
	VisibleTurns   int               `json:"visible_turns,omitempty"`
	TreeLines      []string          `json:"tree_lines,omitempty"`
	Breadcrumb     string            `json:"breadcrumb,omitempty"`
}

// BuildGraphView assembles graph UI state for the active session.
func (s *Session) BuildGraphView(projectRoot string) GraphView {
	out := GraphView{Active: false}
	if s == nil || !s.GraphActive() {
		return out
	}
	bp, _, err := graph.LoadBlueprintByID(projectRoot, s.State.Graph.BlueprintID)
	if err != nil || bp == nil {
		return out
	}
	world := s.State.Graph.World
	cur := strings.TrimSpace(world.CurrentNodeID)
	node, ok := bp.NodeByID(cur)
	if !ok {
		return out
	}
	out.Active = true
	out.GraphID = bp.ID
	out.GraphName = bp.Name
	out.CurrentNodeID = cur
	out.CurrentTitle = node.Title
	out.NodeState = strings.TrimSpace(world.NodeStates[cur])
	out.NodeTools = append([]string(nil), node.ToolsHint...)
	out.ExitHint = "Freie Wahl — graph_complete_node + next_node"
	out.VisibleTurns = VisibleTurnCount(s.State.Messages)
	if len(world.Entities) > 0 {
		out.EntityCount = len(world.Entities)
	}

	for _, e := range bp.Edges {
		if e.To != cur {
			continue
		}
		from, ok := bp.NodeByID(e.From)
		if !ok {
			continue
		}
		out.Predecessors = append(out.Predecessors, GraphViewNode{
			ID:      from.ID,
			Title:   from.Title,
			Label:   e.Label,
			Visited: world.IsVisited(from.ID),
		})
	}
	for _, e := range bp.Edges {
		if e.From != cur {
			continue
		}
		to, ok := bp.NodeByID(e.To)
		if !ok {
			continue
		}
		out.Successors = append(out.Successors, GraphViewNode{
			ID:      to.ID,
			Title:   to.Title,
			Label:   e.Label,
			Visited: world.IsVisited(to.ID),
		})
	}

	party := map[string]bool{}
	for _, id := range world.Party {
		party[strings.TrimSpace(id)] = true
	}
	for _, e := range world.Entities {
		loc := strings.TrimSpace(e.Location)
		ent := GraphViewEntity{
			ID:          e.ID,
			Name:        e.Name,
			Kind:        e.Kind,
			Persistence: e.Persistence,
			Memory:      e.Memory,
			Role:        e.Role,
			TemplateID:  e.TemplateID,
		}
		if loc == cur {
			out.HereEntities = append(out.HereEntities, ent)
		} else if loc == "party" && party[e.ID] {
			out.PartyEntities = append(out.PartyEntities, ent)
		}
	}
	out.Breadcrumb = buildGraphBreadcrumb(bp, world, out.Successors)
	out.TreeLines = buildGraphTreeLines(bp, world)
	return out
}

func buildGraphBreadcrumb(bp *graph.Blueprint, world *graph.WorldState, succ []GraphViewNode) string {
	cur := strings.TrimSpace(world.CurrentNodeID)
	node, ok := bp.NodeByID(cur)
	title := cur
	if ok {
		title = node.Title
	}
	state := strings.TrimSpace(world.NodeStates[cur])
	line := "🗺️ "
	if len(outPredecessors(bp, cur)) > 0 {
		p := outPredecessors(bp, cur)[0]
		line += "[" + p.Title + "] ──▶ "
	}
	line += "🎯 [" + title + "]"
	if state != "" {
		line += " (" + state + ")"
	}
	if len(succ) > 0 {
		line += " ──▶ { "
		for i, s := range succ {
			if i > 0 {
				line += " | "
			}
			line += s.Title
		}
		line += " }"
	}
	return line
}

func outPredecessors(bp *graph.Blueprint, cur string) []graph.Node {
	var out []graph.Node
	for _, e := range bp.Edges {
		if e.To == cur {
			if n, ok := bp.NodeByID(e.From); ok {
				out = append(out, n)
			}
		}
	}
	return out
}

func buildGraphTreeLines(bp *graph.Blueprint, world *graph.WorldState) []string {
	cur := strings.TrimSpace(world.CurrentNodeID)
	party := map[string]bool{}
	for _, id := range world.Party {
		party[strings.TrimSpace(id)] = true
	}
	var lines []string
	for _, n := range bp.Nodes {
		marker := "  "
		if n.ID == cur {
			marker = "◀── "
		}
		vis := ""
		if world.IsVisited(n.ID) && n.ID != cur {
			vis = " (besucht)"
		}
		lines = append(lines, marker+"📍 "+n.Title+vis)
		note := strings.TrimSpace(world.NodeStates[n.ID])
		if note != "" {
			lines = append(lines, "      📝 "+note)
		}
		for _, e := range world.Entities {
			if strings.TrimSpace(e.Location) != n.ID {
				continue
			}
			tag := e.Name
			if e.Persistence == graph.PersistenceStationary {
				tag += " (stationary)"
			}
			lines = append(lines, "      👤 "+tag)
		}
	}
	for _, e := range world.Entities {
		if strings.TrimSpace(e.Location) != "party" || !party[e.ID] {
			continue
		}
		lines = append(lines, "  🎒 "+e.Name+" (party)")
	}
	return lines
}

// MarshalGraphViewJSON serializes the graph panel payload.
func MarshalGraphViewJSON(view GraphView) (string, error) {
	raw, err := json.Marshal(view)
	if err != nil {
		return "{}", err
	}
	return string(raw), nil
}
