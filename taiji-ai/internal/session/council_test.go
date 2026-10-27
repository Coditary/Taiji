package session

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func writeCouncilTemplate(t *testing.T, root, id string) {
	t.Helper()
	dir := filepath.Join(config.RuntimeDir(root), "templates")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	raw := `---
id: ` + id + `
name: Actor One
---
Body`
	if err := os.WriteFile(filepath.Join(dir, id+".md"), []byte(raw), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCouncilEnabledAndActorsForGoal(t *testing.T) {
	if (&Session{}).CouncilEnabled() {
		t.Fatal("empty council disabled")
	}
	s := &Session{State: State{Settings: Settings{
		CouncilEnabled: true,
		CouncilActors:  []string{"a", "b"},
	}}}
	if !s.CouncilEnabled() {
		t.Fatal("expected enabled")
	}
	root := t.TempDir()
	writeCouncilTemplate(t, root, "story")
	actors := CouncilActorsForGoal(s, root, "@story ping")
	if len(actors) != 1 || actors[0] != "story" {
		t.Fatalf("mention actors=%v", actors)
	}
	all := CouncilActorsForGoal(s, root, "hello")
	if len(all) != 2 {
		t.Fatalf("council actors=%v", all)
	}
}

func TestParseCouncilCommandOnOffAddRemove(t *testing.T) {
	root := t.TempDir()
	writeCouncilTemplate(t, root, "story")
	s, err := New(root, "c1", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	reply, handled, err := ParseCouncilCommand(s, root, "/council on")
	if err != nil || !handled || !s.State.Settings.CouncilEnabled {
		t.Fatalf("on: reply=%q err=%v enabled=%v", reply, err, s.State.Settings.CouncilEnabled)
	}
	reply, handled, err = ParseCouncilCommand(s, root, "/council add story")
	if err != nil || !handled || len(s.State.Settings.CouncilActors) != 1 {
		t.Fatalf("add: reply=%q actors=%v err=%v", reply, s.State.Settings.CouncilActors, err)
	}
	reply, handled, err = ParseCouncilCommand(s, root, "/council list")
	if err != nil || !handled || !strings.Contains(reply, "story") {
		t.Fatalf("list: reply=%q err=%v", reply, err)
	}
	reply, handled, err = ParseCouncilCommand(s, root, "/council remove story")
	if err != nil || !handled || len(s.State.Settings.CouncilActors) != 0 {
		t.Fatalf("remove: reply=%q actors=%v err=%v", reply, s.State.Settings.CouncilActors, err)
	}
}

func TestStreamPacingNatural(t *testing.T) {
	s := &Session{State: State{Settings: Settings{StreamPacing: "natural"}}}
	if !s.StreamPacingNatural() {
		t.Fatal("natural pacing")
	}
	s.State.Settings.StreamPacing = "instant"
	if s.StreamPacingNatural() {
		t.Fatal("instant pacing")
	}
}
