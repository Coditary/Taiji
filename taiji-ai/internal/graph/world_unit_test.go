package graph

import "testing"

func minimalBlueprint() *Blueprint {
	return &Blueprint{
		ID:          "test",
		StartNodeID: "room_a",
		Nodes: []Node{
			{ID: "room_a"},
			{ID: "room_b"},
		},
		Edges: []Edge{{From: "room_a", To: "room_b"}},
		Entities: []EntityDef{
			{ID: "npc1", Kind: "npc", Name: "Ada", Location: "room_a"},
			{ID: "npc2", Kind: "npc", Name: "Bob", Location: "party", Persistence: PersistenceUniqueMobile},
		},
	}
}

func TestNewWorldFromBlueprintSkipsEmptyEntityID(t *testing.T) {
	bp := minimalBlueprint()
	bp.Entities = append(bp.Entities, EntityDef{ID: "  ", Location: "room_a"})
	w := NewWorldFromBlueprint(bp)
	if len(w.Entities) != 2 {
		t.Fatalf("entities=%d", len(w.Entities))
	}
}

func TestEntitiesAtPartyMember(t *testing.T) {
	bp := minimalBlueprint()
	w := NewWorldFromBlueprint(bp)
	w.Party = []string{"npc2"}
	at := w.EntitiesAt("room_b")
	if len(at) != 1 || at[0].ID != "npc2" {
		t.Fatalf("at=%#v", at)
	}
}

func TestApplyMutationsUpdateEntityMemory(t *testing.T) {
	bp := minimalBlueprint()
	w := NewWorldFromBlueprint(bp)
	err := w.ApplyMutations(bp, []Mutation{
		{Action: "update_entity", ID: "npc1", Key: "memory", Value: "notes"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if w.Entities["npc1"].Memory != "notes" {
		t.Fatalf("memory=%q", w.Entities["npc1"].Memory)
	}
}

func TestApplyMutationsUnknownEntity(t *testing.T) {
	bp := minimalBlueprint()
	w := NewWorldFromBlueprint(bp)
	err := w.ApplyMutations(bp, []Mutation{{Action: "update_entity", ID: "missing", Key: "memory", Value: "x"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTransitionToRequiresEdge(t *testing.T) {
	bp := minimalBlueprint()
	w := NewWorldFromBlueprint(bp)
	if err := w.TransitionTo(bp, "room_b", true); err != nil {
		t.Fatal(err)
	}
	if w.CurrentNodeID != "room_b" {
		t.Fatalf("node=%q", w.CurrentNodeID)
	}
	if !w.IsVisited("room_a") {
		t.Fatal("room_a should stay in visited history after transition")
	}
}
