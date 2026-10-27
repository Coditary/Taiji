package session

import (
	"testing"

	"github.com/coditary/taiji/internal/graph"
)

func TestGraphGotoRespectsEdges(t *testing.T) {
	bp, err := graph.LoadBlueprint("../../global/graphs/demo-forge/blueprint.yaml")
	if err != nil {
		t.Skip(err)
	}
	s := &Session{State: State{Graph: &GraphSession{
		BlueprintID: bp.ID,
		World:       graph.NewWorldFromBlueprint(bp),
	}}}
	root := "../.."
	if err := GraphGoto(s, root, "node_forest", false); err != nil {
		t.Fatal(err)
	}
	if s.State.Graph.World.CurrentNodeID != "node_forest" {
		t.Fatalf("node = %q", s.State.Graph.World.CurrentNodeID)
	}
	if err := GraphGoto(s, root, "node_forge", false); err != nil {
		t.Fatal(err)
	}
}
