package graph

import (
	"fmt"
	"strings"
)

// NewWorldFromBlueprint creates runtime state from blueprint entity defs.
func NewWorldFromBlueprint(bp *Blueprint) *WorldState {
	w := &WorldState{
		CurrentNodeID: strings.TrimSpace(bp.StartNodeID),
		Entities:      map[string]Entity{},
		NodeStates:    map[string]string{},
	}
	for k, v := range bp.NodeStates {
		w.NodeStates[k] = v
	}
	w.markVisited(w.CurrentNodeID)
	for _, def := range bp.Entities {
		id := strings.TrimSpace(def.ID)
		if id == "" {
			continue
		}
		pers := strings.TrimSpace(def.Persistence)
		if pers == "" {
			pers = PersistenceUniqueMobile
		}
		w.Entities[id] = Entity{
			ID:          id,
			Kind:        strings.TrimSpace(def.Kind),
			Name:        strings.TrimSpace(def.Name),
			Location:    strings.TrimSpace(def.Location),
			Persistence: pers,
			TemplateID:  strings.TrimSpace(def.TemplateID),
			Memory:      strings.TrimSpace(def.Memory),
		}
	}
	return w
}

// EntitiesAt returns entities present at a node or in the traveling party when at node.
func (w *WorldState) EntitiesAt(nodeID string) []Entity {
	if w == nil {
		return nil
	}
	nodeID = strings.TrimSpace(nodeID)
	var out []Entity
	party := map[string]bool{}
	for _, id := range w.Party {
		party[strings.TrimSpace(id)] = true
	}
	for _, e := range w.Entities {
		loc := strings.TrimSpace(e.Location)
		if loc == nodeID {
			out = append(out, e)
			continue
		}
		if loc == "party" && party[e.ID] {
			out = append(out, e)
		}
	}
	return out
}

// ApplyMutations updates world state (entity move, node note, party join).
func (w *WorldState) ApplyMutations(bp *Blueprint, muts []Mutation) error {
	if w == nil {
		return fmt.Errorf("world state is nil")
	}
	for _, m := range muts {
		action := strings.TrimSpace(strings.ToLower(m.Action))
		switch action {
		case "update_entity":
			id := strings.TrimSpace(m.ID)
			if id == "" {
				return fmt.Errorf("update_entity: id required")
			}
			ent, ok := w.Entities[id]
			if !ok {
				return fmt.Errorf("unknown entity %q", id)
			}
			if key := strings.TrimSpace(m.Key); key != "" {
				val := strings.TrimSpace(m.Value)
				switch key {
				case "location":
					if err := w.moveEntity(bp, ent, val); err != nil {
						return err
					}
					ent.Location = val
				case "memory":
					ent.Memory = val
				case "role":
					ent.Role = val
				case "name":
					ent.Name = val
				default:
					return fmt.Errorf("update_entity: unsupported key %q", key)
				}
			}
			if note := strings.TrimSpace(m.Note); note != "" {
				ent.Memory = strings.TrimSpace(ent.Memory + "\n" + note)
			}
			w.Entities[id] = ent
		case "update_node_state":
			id := strings.TrimSpace(m.ID)
			if id == "" {
				return fmt.Errorf("update_node_state: id required")
			}
			if w.NodeStates == nil {
				w.NodeStates = map[string]string{}
			}
			w.NodeStates[id] = strings.TrimSpace(m.Note)
		case "join_party":
			id := strings.TrimSpace(m.ID)
			if id == "" {
				return fmt.Errorf("join_party: id required")
			}
			ent, ok := w.Entities[id]
			if !ok {
				return fmt.Errorf("unknown entity %q", id)
			}
			if ent.Persistence == PersistenceStationary {
				return fmt.Errorf("%s is stationary and cannot join the party", id)
			}
			ent.Location = "party"
			ent.Role = "companion"
			w.Entities[id] = ent
			if !containsString(w.Party, id) {
				w.Party = append(w.Party, id)
			}
		default:
			return fmt.Errorf("unknown mutation action %q", m.Action)
		}
	}
	return nil
}

func (w *WorldState) moveEntity(bp *Blueprint, ent Entity, target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("empty location")
	}
	if ent.Persistence == PersistenceStationary && target != ent.Location {
		return fmt.Errorf("%s cannot leave %s (stationary)", ent.ID, ent.Location)
	}
	if target == "party" {
		if ent.Persistence == PersistenceStationary {
			return fmt.Errorf("%s cannot join party", ent.ID)
		}
		if !containsString(w.Party, ent.ID) {
			w.Party = append(w.Party, ent.ID)
		}
		return nil
	}
	w.Party = removeString(w.Party, ent.ID)
	if bp != nil {
		if _, ok := bp.NodeByID(target); !ok && target != "party" {
			return fmt.Errorf("unknown node %q", target)
		}
	}
	return nil
}

// TransitionTo moves the session to another node if allowed.
func (w *WorldState) TransitionTo(bp *Blueprint, nextNode string, requireEdge bool) error {
	if w == nil || bp == nil {
		return fmt.Errorf("invalid state")
	}
	nextNode = strings.TrimSpace(nextNode)
	if nextNode == "" {
		return fmt.Errorf("next node is required")
	}
	if _, ok := bp.NodeByID(nextNode); !ok {
		return fmt.Errorf("unknown node %q", nextNode)
	}
	cur := strings.TrimSpace(w.CurrentNodeID)
	if requireEdge && cur != "" && cur != nextNode {
		allowed := false
		for _, to := range bp.ExitsFrom(cur) {
			if to == nextNode {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("transition %s -> %s is not allowed", cur, nextNode)
		}
	}
	if cur != "" && cur != nextNode {
		w.markVisited(cur)
	}
	w.CurrentNodeID = nextNode
	return nil
}

func (w *WorldState) markVisited(nodeID string) {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return
	}
	for _, v := range w.VisitedNodes {
		if v == nodeID {
			return
		}
	}
	w.VisitedNodes = append(w.VisitedNodes, nodeID)
}

// IsVisited reports whether a node was entered before.
func (w *WorldState) IsVisited(nodeID string) bool {
	nodeID = strings.TrimSpace(nodeID)
	for _, v := range w.VisitedNodes {
		if v == nodeID {
			return true
		}
	}
	return false
}

func containsString(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}

func removeString(list []string, want string) []string {
	var out []string
	for _, s := range list {
		if s != want {
			out = append(out, s)
		}
	}
	return out
}
