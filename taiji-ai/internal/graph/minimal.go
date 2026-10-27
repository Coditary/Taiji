package graph

// MinimalBlueprint returns a starter graph with one node.
func MinimalBlueprint(id string) *Blueprint {
	return &Blueprint{
		ID:          id,
		Name:        id,
		Description: "Edit this blueprint — nodes are scenes, edges are allowed transitions.",
		StartNodeID: "start",
		Nodes: []Node{
			{
				ID:          "start",
				Title:       "Start",
				Description: "Describe the starting scene here.",
			},
		},
		Edges:      nil,
		Entities:   nil,
		NodeStates: map[string]string{"start": "initial"},
	}
}
