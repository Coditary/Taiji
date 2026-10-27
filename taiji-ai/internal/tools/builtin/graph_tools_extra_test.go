package builtin

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/graph"
	"github.com/coditary/taiji/internal/session"
	"github.com/coditary/taiji/internal/tools"
)

func setupGraphSession(t *testing.T) (string, *session.Session, *graph.Blueprint, *graph.WorldState) {
	t.Helper()
	root := t.TempDir()
	graphID := "story"
	dir := filepath.Join(taijicfg.RuntimeDir(root), "graphs", graphID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bp := graph.MinimalBlueprint(graphID)
	bp.Nodes = append(bp.Nodes, graph.Node{ID: "hall", Title: "Hall"})
	bp.Edges = append(bp.Edges, graph.Edge{From: "start", To: "hall"})
	bp.Entities = []graph.EntityDef{{
		ID: "guide", Kind: "npc", Name: "Guide", Location: "start",
	}}
	bpPath := filepath.Join(dir, "blueprint.yaml")
	if err := graph.SaveBlueprint(bpPath, bp); err != nil {
		t.Fatal(err)
	}
	sess, err := session.New(root, "g-sess", "dummy", "graph play")
	if err != nil {
		t.Fatal(err)
	}
	world := graph.NewWorldFromBlueprint(bp)
	sess.State.Graph = &session.GraphSession{
		BlueprintID:   graphID,
		BlueprintPath: bpPath,
		World:         world,
	}
	return root, sess, bp, world
}

func TestGraphMoveEntityAndJoinParty(t *testing.T) {
	root, sess, _, world := setupGraphSession(t)
	reg := tools.NewRegistry()
	EnableGraphTools(reg, root)
	ctx := session.WithActiveSession(context.Background(), sess)

	out, err := reg.Run(ctx, "graph_move_entity", `{"entity_id":"guide","location":"party","note":"follows the hero"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" || world.Entities["guide"].Location != "party" {
		t.Fatalf("out=%q loc=%q", out.Text, world.Entities["guide"].Location)
	}
}

func TestGraphExtendBlueprintAddsNodes(t *testing.T) {
	root, sess, bp, _ := setupGraphSession(t)
	reg := tools.NewRegistry()
	EnableGraphTools(reg, root)
	ctx := session.WithActiveSession(context.Background(), sess)

	out, err := reg.Run(ctx, "graph_extend_blueprint", `{"nodes":[{"id":"roof","title":"Roof"}],"edges":[{"from":"hall","to":"roof"}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" {
		t.Fatal("expected confirmation text")
	}
	reloaded, err := graph.LoadBlueprint(sess.State.Graph.BlueprintPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(reloaded.Nodes) != len(bp.Nodes)+1 {
		t.Fatalf("nodes = %d want %d", len(reloaded.Nodes), len(bp.Nodes)+1)
	}
}

func TestGraphCompleteNodeMutationsAndForceTransition(t *testing.T) {
	root, sess, _, world := setupGraphSession(t)
	reg := tools.NewRegistry()
	EnableGraphTools(reg, root)
	ctx := session.WithActiveSession(context.Background(), sess)

	_, err := reg.Run(ctx, "graph_complete_node", `{"summary":"done","mutations":[{"action":"update_entity","id":"guide","key":"memory","value":"helpful"}],"next_node":"hall","force_transition":true}`)
	if err != nil {
		t.Fatal(err)
	}
	if world.CurrentNodeID != "hall" {
		t.Fatalf("node = %q", world.CurrentNodeID)
	}
}

func TestParseMutationsAndDecoders(t *testing.T) {
	muts, err := parseMutations([]map[string]any{{"action": "join_party", "id": "x"}})
	if err != nil || len(muts) != 1 || muts[0].Action != "join_party" {
		t.Fatalf("muts = %#v err=%v", muts, err)
	}
	nodes, err := decodeNodeList([]map[string]any{{"id": "n1", "title": "N"}})
	if err != nil || len(nodes) != 1 {
		t.Fatalf("nodes = %#v err=%v", nodes, err)
	}
	edges, err := decodeEdgeList([]map[string]any{{"from": "a", "to": "b"}})
	if err != nil || len(edges) != 1 {
		t.Fatalf("edges = %#v err=%v", edges, err)
	}
	ents, err := decodeEntityDefList([]map[string]any{{"id": "e1", "location": "start"}})
	if err != nil || len(ents) != 1 {
		t.Fatalf("entities = %#v err=%v", ents, err)
	}
	if !boolArg(map[string]any{"force_transition": true}, "force_transition") {
		t.Fatal("expected true boolArg")
	}
}

func TestEnableGraphToolsNilRegistry(t *testing.T) {
	EnableGraphTools(nil, t.TempDir())
}
