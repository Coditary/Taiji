package session

import (
	"os"
	"path/filepath"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/graph"
)

func TestGraphSessionActiveAndContext(t *testing.T) {
	root := t.TempDir()
	graphID := "g1"
	dir := filepath.Join(taijicfg.RuntimeDir(root), "graphs", graphID)
	_ = os.MkdirAll(dir, 0o755)
	bp := graph.MinimalBlueprint(graphID)
	_ = graph.SaveBlueprint(filepath.Join(dir, "blueprint.yaml"), bp)

	s := &Session{State: State{Graph: &GraphSession{
		BlueprintID: graphID,
		World:       graph.NewWorldFromBlueprint(bp),
	}}}
	if !s.GraphActive() {
		t.Fatal("expected active graph")
	}
	ctx := s.BuildGraphContextAppendix(root)
	if ctx == "" {
		t.Fatal("expected context appendix")
	}
	s.EnsureGraphToolsAllowed()
}

func TestGraphActiveFalseWhenMissingWorld(t *testing.T) {
	s := &Session{State: State{Graph: &GraphSession{BlueprintID: "x"}}}
	if s.GraphActive() {
		t.Fatal("expected inactive without world")
	}
}
