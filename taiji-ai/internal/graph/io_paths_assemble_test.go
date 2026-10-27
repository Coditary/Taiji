package graph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestMinimalBlueprintValidates(t *testing.T) {
	bp := MinimalBlueprint("demo")
	if err := bp.Validate(); err != nil {
		t.Fatal(err)
	}
	if bp.StartNodeID != "start" || len(bp.Nodes) != 1 {
		t.Fatalf("bp=%+v", bp)
	}
}

func TestSaveLoadBlueprintRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "graphs", "g1", "blueprint.yaml")
	bp := MinimalBlueprint("g1")
	bp.Name = "Graph One"
	bp.Nodes[0].Title = "Opening"
	if err := SaveBlueprint(path, bp); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBlueprint(path)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.ID != "g1" || loaded.Nodes[0].Title != "Opening" {
		t.Fatalf("loaded=%+v", loaded)
	}
}

func TestLoadBlueprintByIDFromRuntimeDir(t *testing.T) {
	root := t.TempDir()
	graphID := "adventure"
	dir := filepath.Join(taijicfg.RuntimeDir(root), "graphs", graphID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := SaveBlueprint(filepath.Join(dir, "blueprint.yaml"), MinimalBlueprint(graphID)); err != nil {
		t.Fatal(err)
	}
	bp, path, err := LoadBlueprintByID(root, graphID)
	if err != nil {
		t.Fatal(err)
	}
	if bp.ID != graphID || !strings.HasSuffix(path, "blueprint.yaml") {
		t.Fatalf("bp=%+v path=%q", bp, path)
	}
}

func TestExtendBlueprintAddsNodeAndEdge(t *testing.T) {
	bp := MinimalBlueprint("x")
	if err := ExtendBlueprint(bp, []Node{{ID: "hall", Title: "Hall"}}, []Edge{{From: "start", To: "hall"}}, nil); err != nil {
		t.Fatal(err)
	}
	if len(bp.Nodes) != 2 || len(bp.Edges) != 1 {
		t.Fatalf("nodes=%d edges=%d", len(bp.Nodes), len(bp.Edges))
	}
}

func TestAssembleContextListsExitsAndEntities(t *testing.T) {
	bp := minimalBlueprint()
	bp.Name = "Test Graph"
	bp.Nodes[0].Title = "Room A"
	bp.Nodes[0].Description = "A dusty room."
	bp.Edges[0].Label = "door"
	w := NewWorldFromBlueprint(bp)
	ctx := AssembleContext(bp, w)
	for _, want := range []string{"Test Graph", "Room A", "room_b", "door", "Ada"} {
		if !strings.Contains(ctx, want) {
			t.Fatalf("missing %q in:\n%s", want, ctx)
		}
	}
}

func TestWorldMoveEntityPartyAndStationary(t *testing.T) {
	bp := minimalBlueprint()
	bp.Entities = append(bp.Entities, EntityDef{
		ID: "statue", Kind: "prop", Name: "Statue", Location: "room_a", Persistence: PersistenceStationary,
	})
	w := NewWorldFromBlueprint(bp)
	if err := w.ApplyMutations(bp, []Mutation{{Action: "join_party", ID: "npc2"}}); err != nil {
		t.Fatal(err)
	}
	if len(w.Party) != 1 {
		t.Fatalf("party=%v", w.Party)
	}
	err := w.ApplyMutations(bp, []Mutation{{Action: "update_entity", ID: "statue", Key: "location", Value: "room_b"}})
	if err == nil {
		t.Fatal("stationary entity should not leave room_a")
	}
}

func TestTransitionToRequiresEdgeWhenFlagSet(t *testing.T) {
	bp := minimalBlueprint()
	w := NewWorldFromBlueprint(bp)
	if err := w.TransitionTo(bp, "room_b", true); err != nil {
		t.Fatal(err)
	}
	if err := w.TransitionTo(bp, "room_a", true); err == nil {
		t.Fatal("reverse transition without edge should fail")
	}
}
