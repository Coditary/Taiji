package graph

import "testing"

func TestBardinLeavesForgeNoDuplicate(t *testing.T) {
	bp, err := LoadBlueprint("../../../global/graphs/demo-forge/blueprint.yaml")
	if err != nil {
		t.Skip(err)
	}
	world := NewWorldFromBlueprint(bp)
	atForge := world.EntitiesAt("node_forge")
	if len(atForge) != 1 || atForge[0].ID != "npc_bardin" {
		t.Fatalf("forge entities = %#v", atForge)
	}
	err = world.ApplyMutations(bp, []Mutation{
		{Action: "join_party", ID: "npc_bardin"},
		{Action: "update_node_state", ID: "node_forge", Note: "Verwaist. Bardin ist mit dem Spieler aufgebrochen."},
	})
	if err != nil {
		t.Fatal(err)
	}
	atForge = world.EntitiesAt("node_forge")
	if len(atForge) != 0 {
		t.Fatalf("bardin should not be at forge, got %#v", atForge)
	}
	err = world.TransitionTo(bp, "node_forest", true)
	if err != nil {
		t.Fatal(err)
	}
	withPlayer := world.EntitiesAt("node_forest")
	found := false
	for _, e := range withPlayer {
		if e.ID == "npc_bardin" {
			found = true
		}
	}
	if !found {
		t.Fatalf("bardin should travel with party in forest, got %#v", withPlayer)
	}
}
