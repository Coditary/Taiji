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

func TestGraphCompleteNodeTransitions(t *testing.T) {
	root := t.TempDir()
	graphID := "demo"
	dir := filepath.Join(taijicfg.RuntimeDir(root), "graphs", graphID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bp := graph.MinimalBlueprint(graphID)
	bp.Nodes = append(bp.Nodes, graph.Node{ID: "hall", Title: "Hall"})
	bp.Edges = append(bp.Edges, graph.Edge{From: "start", To: "hall"})
	if err := graph.SaveBlueprint(filepath.Join(dir, "blueprint.yaml"), bp); err != nil {
		t.Fatal(err)
	}

	sess, err := session.New(root, "", "dummy", "graph")
	if err != nil {
		t.Fatal(err)
	}
	world := graph.NewWorldFromBlueprint(bp)
	sess.State.Graph = &session.GraphSession{BlueprintID: graphID, World: world}

	reg := tools.NewRegistry()
	EnableGraphTools(reg, root)
	ctx := session.WithActiveSession(context.Background(), sess)

	out, err := reg.Run(ctx, "graph_complete_node", `{"summary":"Opened the door","next_node":"hall"}`)
	if err != nil {
		t.Fatal(err)
	}
	if out.Text == "" || world.CurrentNodeID != "hall" {
		t.Fatalf("world node=%q out=%q", world.CurrentNodeID, out.Text)
	}
}

func TestGraphMoveEntityRequiresSession(t *testing.T) {
	reg := tools.NewRegistry()
	EnableGraphTools(reg, t.TempDir())
	_, err := reg.Run(context.Background(), "graph_move_entity", `{"entity_id":"npc","location":"party"}`)
	if err == nil {
		t.Fatal("expected error without active session")
	}
}
