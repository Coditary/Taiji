package graph

import (
	"fmt"
	"strings"
)

// AssembleContext builds the runtime system appendix for the current node.
func AssembleContext(bp *Blueprint, world *WorldState) string {
	if bp == nil || world == nil {
		return ""
	}
	nodeID := strings.TrimSpace(world.CurrentNodeID)
	node, ok := bp.NodeByID(nodeID)
	if !ok {
		return ""
	}
	var b strings.Builder
	b.WriteString("[Guided graph]\n")
	b.WriteString(fmt.Sprintf("Graph: %s (%s)\n", bp.Name, bp.ID))
	b.WriteString(fmt.Sprintf("Current node: %s — %s\n", node.ID, node.Title))
	if desc := strings.TrimSpace(node.Description); desc != "" {
		b.WriteString(desc)
		b.WriteString("\n")
	}
	if note := strings.TrimSpace(world.NodeStates[nodeID]); note != "" {
		b.WriteString(fmt.Sprintf("Node state: %s\n", note))
	}
	exits := bp.ExitsFrom(nodeID)
	if len(exits) > 0 {
		b.WriteString("Allowed exits (use graph_complete_node with next_node):\n")
		for _, e := range bp.Edges {
			if e.From != nodeID {
				continue
			}
			label := strings.TrimSpace(e.Label)
			if label != "" {
				b.WriteString(fmt.Sprintf("  - %s (%s)\n", e.To, label))
			} else {
				b.WriteString(fmt.Sprintf("  - %s\n", e.To))
			}
		}
	}
	present := world.EntitiesAt(nodeID)
	if len(present) > 0 {
		b.WriteString("Entities here (or in party):\n")
		for _, e := range present {
			line := fmt.Sprintf("  - %s (%s)", e.Name, e.ID)
			if e.TemplateID != "" {
				line += fmt.Sprintf(" template=%s", e.TemplateID)
			}
			if e.Role != "" {
				line += fmt.Sprintf(" role=%s", e.Role)
			}
			if e.Location == "party" {
				line += " [traveling with player]"
			}
			if mem := strings.TrimSpace(e.Memory); mem != "" {
				line += fmt.Sprintf(" — %s", mem)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}
	b.WriteString("\nUse graph_move_entity / graph_complete_node to mutate world state. ")
	b.WriteString("Do not invent duplicate NPCs; only entities listed above exist unless you add them via graph_extend_blueprint.\n")
	return strings.TrimSpace(b.String())
}
