package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/graph"
	"github.com/coditary/taiji/internal/provider"
)

func TestBuildGraphViewInactive(t *testing.T) {
	var nilSess *Session
	if nilSess.BuildGraphView("").Active {
		t.Fatal("nil session")
	}
	s := &Session{}
	if s.BuildGraphView("").Active {
		t.Fatal("no graph")
	}
}

func TestBuildGraphViewPopulatesRadar(t *testing.T) {
	root := t.TempDir()
	graphID := "demo"
	dir := filepath.Join(taijicfg.RuntimeDir(root), "graphs", graphID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bp := &graph.Blueprint{
		ID:          graphID,
		Name:        "Demo graph",
		StartNodeID: "start",
		Nodes: []graph.Node{
			{ID: "start", Title: "Start scene"},
			{ID: "next", Title: "Next scene"},
		},
		Edges: []graph.Edge{
			{From: "start", To: "next", Label: "door"},
			{From: "next", To: "start", Label: "back"},
		},
		Entities: []graph.EntityDef{
			{ID: "hero", Name: "Hero", Kind: "pc", Location: "start"},
			{ID: "bag", Name: "Bag", Kind: "item", Location: "party"},
		},
		NodeStates: map[string]string{"start": "calm"},
	}
	if err := graph.SaveBlueprint(filepath.Join(dir, "blueprint.yaml"), bp); err != nil {
		t.Fatal(err)
	}
	world := graph.NewWorldFromBlueprint(bp)
	world.Party = []string{"bag"}
	world.NodeStates["start"] = "calm"

	s := &Session{
		State: State{
			Messages: []provider.Message{
				{Role: provider.RoleUser, Content: "hello"},
				{Role: provider.RoleAssistant, Content: "hi"},
			},
			Graph: &GraphSession{BlueprintID: graphID, World: world},
		},
	}

	view := s.BuildGraphView(root)
	if !view.Active {
		t.Fatal("expected active graph view")
	}
	if view.GraphName != "Demo graph" || view.CurrentNodeID != "start" {
		t.Fatalf("view=%+v", view)
	}
	if len(view.Successors) != 1 || view.Successors[0].ID != "next" {
		t.Fatalf("successors=%+v", view.Successors)
	}
	if len(view.HereEntities) != 1 || view.HereEntities[0].Name != "Hero" {
		t.Fatalf("here=%+v", view.HereEntities)
	}
	if len(view.PartyEntities) != 1 || view.PartyEntities[0].Name != "Bag" {
		t.Fatalf("party=%+v", view.PartyEntities)
	}
	if view.Breadcrumb == "" || !strings.Contains(view.Breadcrumb, "Start scene") {
		t.Fatalf("breadcrumb=%q", view.Breadcrumb)
	}
	if len(view.TreeLines) == 0 {
		t.Fatal("expected tree lines")
	}

	raw, err := MarshalGraphViewJSON(view)
	if err != nil || !strings.Contains(raw, `"active":true`) {
		t.Fatalf("json=%q err=%v", raw, err)
	}
}
