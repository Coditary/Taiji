package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/graph"
)

func setupGraphBlueprint(t *testing.T, root, graphID string) {
	t.Helper()
	dir := filepath.Join(taijicfg.RuntimeDir(root), "graphs", graphID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	bp := graph.MinimalBlueprint(graphID)
	if err := graph.SaveBlueprint(filepath.Join(dir, "blueprint.yaml"), bp); err != nil {
		t.Fatal(err)
	}
}

func TestParseGraphCommandNilSession(t *testing.T) {
	reply, handled, err := ParseGraphCommand(nil, t.TempDir(), "/graph")
	if err != nil || handled || reply != "" {
		t.Fatalf("handled=%v reply=%q err=%v", handled, reply, err)
	}
}

func TestParseGraphCommandStatusAndCatalog(t *testing.T) {
	root := t.TempDir()
	setupGraphBlueprint(t, root, "g1")
	s, err := New(root, "sess", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseGraphCommand(s, root, "/graph")
	if err != nil || !handled || !strings.Contains(reply, "Graph-Befehle") {
		t.Fatalf("status: handled=%v reply=%q err=%v", handled, reply, err)
	}
	reply, handled, err = ParseGraphCommand(s, root, "/graph list")
	if err != nil || !handled || !strings.Contains(reply, "g1") {
		t.Fatalf("list: reply=%q err=%v", reply, err)
	}
}

func TestParseGraphCommandLoadUnload(t *testing.T) {
	root := t.TempDir()
	setupGraphBlueprint(t, root, "g1")
	s, err := New(root, "sess", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseGraphCommand(s, root, "/graph load g1")
	if err != nil || !handled || !strings.Contains(reply, "Graph geladen") {
		t.Fatalf("load: reply=%q err=%v", reply, err)
	}
	if !s.GraphActive() {
		t.Fatal("expected active graph")
	}
	reply, handled, err = ParseGraphCommand(s, root, "/graph unload")
	if err != nil || !handled || s.State.Graph != nil {
		t.Fatalf("unload: reply=%q graph=%v err=%v", reply, s.State.Graph, err)
	}
}

func TestParseGraphCommandNewBlueprint(t *testing.T) {
	root := t.TempDir()
	s, err := New(root, "sess", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseGraphCommand(s, root, "/graph new fresh")
	if err != nil || !handled || !strings.Contains(reply, "Blueprint erstellt") {
		t.Fatalf("new: reply=%q err=%v", reply, err)
	}
	_, handled, err = ParseGraphCommand(s, root, "/graph new fresh")
	if err == nil || !handled {
		t.Fatalf("duplicate new: err=%v handled=%v", err, handled)
	}
}

func TestParseGraphCommandResetAndSave(t *testing.T) {
	root := t.TempDir()
	setupGraphBlueprint(t, root, "g1")
	s, err := New(root, "sess", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ParseGraphCommand(s, root, "/graph load g1"); err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseGraphCommand(s, root, "/graph save")
	if err != nil || !handled || !strings.Contains(reply, "gespeichert") {
		t.Fatalf("save: reply=%q err=%v", reply, err)
	}
	reply, handled, err = ParseGraphCommand(s, root, "/graph reset")
	if err != nil || !handled || !strings.Contains(reply, "zurückgesetzt") {
		t.Fatalf("reset: reply=%q err=%v", reply, err)
	}
}

func TestParseGraphCommandNotHandled(t *testing.T) {
	s, err := New(t.TempDir(), "sess", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	_, handled, err := ParseGraphCommand(s, t.TempDir(), "/not-graph")
	if err != nil || handled {
		t.Fatalf("handled=%v err=%v", handled, err)
	}
}
