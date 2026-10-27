package graph

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadBlueprint reads blueprint.yaml from disk.
func LoadBlueprint(path string) (*Blueprint, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bp Blueprint
	if err := yaml.Unmarshal(raw, &bp); err != nil {
		return nil, err
	}
	if err := bp.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(bp.ID) == "" {
		bp.ID = strings.TrimSuffix(filepath.Base(filepath.Dir(path)), "")
	}
	return &bp, nil
}

// SaveBlueprint writes blueprint.yaml (creates parent dirs).
func SaveBlueprint(path string, bp *Blueprint) error {
	if bp == nil {
		return fmt.Errorf("blueprint is nil")
	}
	if err := bp.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	raw, err := yaml.Marshal(bp)
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o644)
}

// LoadBlueprintByID resolves and loads a graph blueprint.
func LoadBlueprintByID(projectRoot, graphID string) (*Blueprint, string, error) {
	path, err := ResolveBlueprintPath(projectRoot, graphID)
	if err != nil {
		return nil, "", err
	}
	bp, err := LoadBlueprint(path)
	if err != nil {
		return nil, "", err
	}
	return bp, path, nil
}

func (bp *Blueprint) Validate() error {
	if bp == nil {
		return fmt.Errorf("blueprint is nil")
	}
	if strings.TrimSpace(bp.StartNodeID) == "" {
		return fmt.Errorf("start_node is required")
	}
	if len(bp.Nodes) == 0 {
		return fmt.Errorf("at least one node is required")
	}
	ids := map[string]bool{}
	for _, n := range bp.Nodes {
		id := strings.TrimSpace(n.ID)
		if id == "" {
			return fmt.Errorf("node id is required")
		}
		ids[id] = true
	}
	if !ids[bp.StartNodeID] {
		return fmt.Errorf("start_node %q not found", bp.StartNodeID)
	}
	for _, e := range bp.Edges {
		if !ids[e.From] || !ids[e.To] {
			return fmt.Errorf("edge %s -> %s references unknown node", e.From, e.To)
		}
	}
	return nil
}

// NodeByID returns a node definition.
func (bp *Blueprint) NodeByID(id string) (Node, bool) {
	id = strings.TrimSpace(id)
	for _, n := range bp.Nodes {
		if n.ID == id {
			return n, true
		}
	}
	return Node{}, false
}

// ExitsFrom lists allowed target node ids from a node.
func (bp *Blueprint) ExitsFrom(from string) []string {
	from = strings.TrimSpace(from)
	var out []string
	for _, e := range bp.Edges {
		if e.From == from {
			out = append(out, e.To)
		}
	}
	return out
}
