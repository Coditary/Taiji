package graph

import (
	"fmt"
	"strings"
)

// ExtendBlueprint applies additive edits (nodes/edges/entities) from the model.
func ExtendBlueprint(bp *Blueprint, addNodes []Node, addEdges []Edge, addEntities []EntityDef) error {
	if bp == nil {
		return fmt.Errorf("blueprint is nil")
	}
	ids := map[string]bool{}
	for _, n := range bp.Nodes {
		ids[n.ID] = true
	}
	for _, n := range addNodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			return fmt.Errorf("node id required")
		}
		if ids[id] {
			return fmt.Errorf("node %q already exists", id)
		}
		if strings.TrimSpace(n.Title) == "" {
			n.Title = id
		}
		bp.Nodes = append(bp.Nodes, n)
		ids[id] = true
	}
	for _, e := range addEdges {
		if !ids[e.From] || !ids[e.To] {
			return fmt.Errorf("edge %s -> %s references unknown node", e.From, e.To)
		}
		bp.Edges = append(bp.Edges, e)
	}
	entIDs := map[string]bool{}
	for _, d := range bp.Entities {
		entIDs[d.ID] = true
	}
	for _, d := range addEntities {
		id := strings.TrimSpace(d.ID)
		if id == "" {
			continue
		}
		if entIDs[id] {
			return fmt.Errorf("entity %q already exists in blueprint", id)
		}
		bp.Entities = append(bp.Entities, d)
		entIDs[id] = true
	}
	return bp.Validate()
}
