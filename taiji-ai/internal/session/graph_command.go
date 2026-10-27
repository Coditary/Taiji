package session

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/coditary/taiji/internal/graph"
)

// ParseGraphCommand handles /graph subcommands.
func ParseGraphCommand(s *Session, projectRoot, line string) (reply string, handled bool, err error) {
	if s == nil {
		return "", false, nil
	}
	line = strings.TrimSpace(line)
	lower := strings.ToLower(line)
	if lower == "/graph" {
		return formatGraphStatus(s, projectRoot), true, nil
	}
	if !strings.HasPrefix(lower, "/graph ") {
		return "", false, nil
	}
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return formatGraphStatus(s, projectRoot), true, nil
	}
	sub := strings.ToLower(fields[1])
	switch sub {
	case "status", "list":
		if sub == "list" {
			return formatGraphCatalog(projectRoot), true, nil
		}
		return formatGraphStatus(s, projectRoot), true, nil
	case "load":
		if len(fields) < 3 {
			return "Nutze: /graph load <graph-id>", true, nil
		}
		id := strings.TrimSpace(fields[2])
		if err := LoadGraphSession(s, projectRoot, id, true); err != nil {
			return "", true, err
		}
		s.EnsureGraphToolsAllowed()
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Graph geladen: %s\n%s", id, s.BuildGraphContextAppendix(projectRoot)), true, nil
	case "unload", "off":
		s.State.Graph = nil
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return "Graph-Modus beendet.", true, nil
	case "reset":
		if !s.GraphActive() {
			return "Kein Graph aktiv.", true, nil
		}
		id := s.State.Graph.BlueprintID
		if err := LoadGraphSession(s, projectRoot, id, true); err != nil {
			return "", true, err
		}
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return "Weltzustand zurückgesetzt (Blueprint-Initialzustand).", true, nil
	case "save":
		if !s.GraphActive() {
			return "Kein Graph aktiv — Welt lebt in state.json.", true, nil
		}
		if err := s.Save(); err != nil {
			return "", true, err
		}
		return "Session-Weltzustand gespeichert.", true, nil
	case "new":
		if len(fields) < 3 {
			return "Nutze: /graph new <graph-id>", true, nil
		}
		id := strings.TrimSpace(fields[2])
		dir := graph.BlueprintDirForWrite(projectRoot, id)
		path := filepath.Join(dir, "blueprint.yaml")
		if _, err := os.Stat(path); err == nil {
			return "", true, fmt.Errorf("graph %q existiert bereits", id)
		}
		bp := graph.MinimalBlueprint(id)
		if err := graph.SaveBlueprint(path, bp); err != nil {
			return "", true, err
		}
		return fmt.Sprintf("Blueprint erstellt: %s\nBearbeite %s und dann /graph load %s", id, path, id), true, nil
	default:
		return formatGraphHelp(), true, nil
	}
}

// LoadGraphSession attaches a blueprint and fresh or preserved world.
func LoadGraphSession(s *Session, projectRoot, graphID string, resetWorld bool) error {
	bp, path, err := graph.LoadBlueprintByID(projectRoot, graphID)
	if err != nil {
		return err
	}
	gs := s.State.Graph
	if gs == nil || resetWorld || gs.BlueprintID != graphID {
		gs = &GraphSession{
			BlueprintID:   bp.ID,
			BlueprintPath: path,
			World:         graph.NewWorldFromBlueprint(bp),
		}
	} else {
		gs.BlueprintPath = path
		if gs.World == nil {
			gs.World = graph.NewWorldFromBlueprint(bp)
		}
	}
	s.State.Graph = gs
	return nil
}

func formatGraphHelp() string {
	return strings.TrimSpace(`Graph-Befehle:
  /graph list          — verfügbare Blueprints
  /graph load <id>     — Blueprint + Welt in Session laden
  /graph status        — aktueller Knoten & Entities
  /graph new <id>      — leeres Blueprint unter .taiji/graphs/
  /graph reset         — Welt auf Initialzustand
  /graph save          — Session speichern
  /graph unload        — Graph-Modus aus`)
}

func formatGraphCatalog(projectRoot string) string {
	var b strings.Builder
	b.WriteString("Graph blueprints:\n")
	found := false
	for _, dir := range graph.GraphDirs(projectRoot) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, ent := range entries {
			if !ent.IsDir() {
				continue
			}
			bpPath := filepath.Join(dir, ent.Name(), "blueprint.yaml")
			if _, err := os.Stat(bpPath); err != nil {
				continue
			}
			found = true
			b.WriteString(fmt.Sprintf("  - %s (%s)\n", ent.Name(), bpPath))
		}
	}
	if !found {
		b.WriteString("  (keine — /graph new <id> oder global/graphs/)\n")
	}
	b.WriteString("\n")
	b.WriteString(formatGraphHelp())
	return strings.TrimSpace(b.String())
}

func formatGraphStatus(s *Session, projectRoot string) string {
	if !s.GraphActive() {
		return "Kein Graph aktiv.\n" + formatGraphHelp()
	}
	appendix := s.BuildGraphContextAppendix(projectRoot)
	if appendix == "" {
		return fmt.Sprintf("Graph %s aktiv (Kontext konnte nicht geladen werden).", s.State.Graph.BlueprintID)
	}
	return appendix + "\n\n" + formatGraphHelp()
}
