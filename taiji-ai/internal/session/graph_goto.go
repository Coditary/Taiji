package session

import (
	"fmt"
	"strings"

	"github.com/coditary/taiji/internal/graph"
)

// GraphGoto transitions the session to a node (validates edges unless force).
func GraphGoto(s *Session, projectRoot, nodeID string, force bool) error {
	if s == nil || !s.GraphActive() {
		return fmt.Errorf("no active graph")
	}
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("node id is required")
	}
	bp, _, err := graph.LoadBlueprintByID(projectRoot, s.State.Graph.BlueprintID)
	if err != nil {
		return err
	}
	world := s.State.Graph.World
	if err := world.TransitionTo(bp, nodeID, !force); err != nil {
		return err
	}
	return s.Save()
}
