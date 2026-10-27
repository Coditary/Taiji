package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/graph"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

// GraphTools implements guided-graph world mutations (project root from runtime).
type GraphTools struct {
	ProjectRoot string
}

// EnableGraphTools registers graph_* builtins.
func EnableGraphTools(reg *tools.Registry, projectRoot string) {
	if reg == nil {
		return
	}
	g := GraphTools{ProjectRoot: strings.TrimSpace(projectRoot)}
	reg.Register(g.completeNodeTool())
	reg.Register(g.moveEntityTool())
	reg.Register(g.extendBlueprintTool())
}

func (g GraphTools) load(sess *session.Session) (*graph.Blueprint, *graph.WorldState, error) {
	if sess == nil || !sess.GraphActive() {
		return nil, nil, fmt.Errorf("no active graph in session — use /graph load <id>")
	}
	bp, _, err := graph.LoadBlueprintByID(g.ProjectRoot, sess.State.Graph.BlueprintID)
	if err != nil {
		return nil, nil, err
	}
	return bp, sess.State.Graph.World, nil
}

func (g GraphTools) completeNodeTool() tools.Tool {
	return toolDef{
		name: "graph_complete_node",
		desc: "Mark work in the current graph node done, apply world mutations, and optionally move to next_node (must be an allowed edge unless force_transition).",
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"summary":           map[string]any{"type": "string", "description": "Short note what was completed in this node"},
				"next_node":         map[string]any{"type": "string", "description": "Target node id (optional)"},
				"force_transition":  map[string]any{"type": "boolean", "description": "Skip edge validation (GM override)"},
				"mutations":         map[string]any{"type": "array", "description": "World mutations (update_entity, update_node_state, join_party)"},
			},
		},
		run: func(ctx context.Context, args map[string]any) (string, error) {
			sess, ok := session.ActiveSessionFrom(ctx)
			if !ok {
				return "", fmt.Errorf("no active session")
			}
			bp, world, err := g.load(sess)
			if err != nil {
				return "", err
			}
			var muts []graph.Mutation
			if raw, ok := args["mutations"]; ok {
				muts, err = parseMutations(raw)
				if err != nil {
					return "", err
				}
			}
			if summary := strings.TrimSpace(stringArg(args, "summary")); summary != "" {
				cur := strings.TrimSpace(world.CurrentNodeID)
				muts = append(muts, graph.Mutation{Action: "update_node_state", ID: cur, Note: summary})
			}
			if err := world.ApplyMutations(bp, muts); err != nil {
				return "", err
			}
			next := strings.TrimSpace(stringArg(args, "next_node"))
			if next != "" {
				force := boolArg(args, "force_transition")
				if err := world.TransitionTo(bp, next, !force); err != nil {
					return "", err
				}
			}
			if err := sess.Save(); err != nil {
				return "", err
			}
			msg := graph.AssembleContext(bp, world)
			return "ok\n" + msg, nil
		},
	}
}

func (g GraphTools) moveEntityTool() tools.Tool {
	return toolDef{
		name: "graph_move_entity",
		desc: "Move a unique entity to a node id or 'party'. Respects stationary persistence.",
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"entity_id": map[string]any{"type": "string"},
				"location":  map[string]any{"type": "string"},
				"note":      map[string]any{"type": "string", "description": "Optional memory append"},
			},
			"required": []string{"entity_id", "location"},
		},
		run: func(ctx context.Context, args map[string]any) (string, error) {
			sess, ok := session.ActiveSessionFrom(ctx)
			if !ok {
				return "", fmt.Errorf("no active session")
			}
			bp, world, err := g.load(sess)
			if err != nil {
				return "", err
			}
			id := strings.TrimSpace(stringArg(args, "entity_id"))
			loc := strings.TrimSpace(stringArg(args, "location"))
			muts := []graph.Mutation{
				{Action: "update_entity", ID: id, Key: "location", Value: loc},
			}
			if note := strings.TrimSpace(stringArg(args, "note")); note != "" {
				muts[0].Note = note
			}
			if loc == "party" {
				muts = append(muts, graph.Mutation{Action: "join_party", ID: id})
			}
			if err := world.ApplyMutations(bp, muts); err != nil {
				return "", err
			}
			if err := sess.Save(); err != nil {
				return "", err
			}
			return fmt.Sprintf("%s -> %s", id, loc), nil
		},
	}
}

func (g GraphTools) extendBlueprintTool() tools.Tool {
	return toolDef{
		name: "graph_extend_blueprint",
		desc: "Add nodes, edges, or entity definitions to the active graph blueprint (persisted under .taiji/graphs/).",
		params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"nodes":    map[string]any{"type": "array"},
				"edges":    map[string]any{"type": "array"},
				"entities": map[string]any{"type": "array"},
			},
		},
		run: func(ctx context.Context, args map[string]any) (string, error) {
			sess, ok := session.ActiveSessionFrom(ctx)
			if !ok {
				return "", fmt.Errorf("no active session")
			}
			if !sess.GraphActive() {
				return "", fmt.Errorf("no active graph")
			}
			path := strings.TrimSpace(sess.State.Graph.BlueprintPath)
			if path == "" {
				path, err := graph.ResolveBlueprintPath(g.ProjectRoot, sess.State.Graph.BlueprintID)
				if err != nil {
					return "", err
				}
				sess.State.Graph.BlueprintPath = path
			}
			bp, err := graph.LoadBlueprint(path)
			if err != nil {
				return "", err
			}
			addNodes, _ := decodeNodeList(args["nodes"])
			addEdges, _ := decodeEdgeList(args["edges"])
			addEnts, _ := decodeEntityDefList(args["entities"])
			if err := graph.ExtendBlueprint(bp, addNodes, addEdges, addEnts); err != nil {
				return "", err
			}
			if err := graph.SaveBlueprint(path, bp); err != nil {
				return "", err
			}
			return fmt.Sprintf("blueprint extended (%d nodes, %d edges, %d entities added)", len(addNodes), len(addEdges), len(addEnts)), nil
		},
	}
}

type toolDef struct {
	name   string
	desc   string
	params map[string]any
	run    func(context.Context, map[string]any) (string, error)
}

func (t toolDef) Name() string        { return t.name }
func (t toolDef) Description() string { return t.desc }
func (t toolDef) Parameters() map[string]any { return t.params }
func (t toolDef) Run(ctx context.Context, args map[string]any) (string, error) {
	return t.run(ctx, args)
}

func parseMutations(raw any) ([]graph.Mutation, error) {
	data, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var muts []graph.Mutation
	if err := json.Unmarshal(data, &muts); err != nil {
		return nil, err
	}
	return muts, nil
}

func boolArg(args map[string]any, key string) bool {
	v, ok := args[key]
	if !ok {
		return false
	}
	switch b := v.(type) {
	case bool:
		return b
	default:
		return false
	}
}

func decodeNodeList(raw any) ([]graph.Node, error) {
	if raw == nil {
		return nil, nil
	}
	data, _ := json.Marshal(raw)
	var out []graph.Node
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeEdgeList(raw any) ([]graph.Edge, error) {
	if raw == nil {
		return nil, nil
	}
	data, _ := json.Marshal(raw)
	var out []graph.Edge
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func decodeEntityDefList(raw any) ([]graph.EntityDef, error) {
	if raw == nil {
		return nil, nil
	}
	data, _ := json.Marshal(raw)
	var out []graph.EntityDef
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
