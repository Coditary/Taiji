package graph

// Persistence controls how entities behave when scenes change.
const (
	PersistenceStationary    = "stationary"
	PersistenceUniqueMobile  = "unique_mobile"
	PersistenceInstanced     = "instanced"
)

// Blueprint is the static topology (design-time).
type Blueprint struct {
	ID          string            `yaml:"id"`
	Name        string            `yaml:"name"`
	Description string            `yaml:"description,omitempty"`
	StartNodeID string            `yaml:"start_node"`
	Nodes       []Node            `yaml:"nodes"`
	Edges       []Edge            `yaml:"edges"`
	Entities    []EntityDef       `yaml:"entities,omitempty"`
	NodeStates  map[string]string `yaml:"initial_node_states,omitempty"`
}

// Node describes a scene/phase (environment only — no hard-wired NPCs at runtime).
type Node struct {
	ID          string   `yaml:"id"`
	Title       string   `yaml:"title"`
	Description string   `yaml:"description"`
	ToolsHint   []string `yaml:"tools_hint,omitempty"`
}

// Edge is an allowed transition from -> to.
type Edge struct {
	From  string `yaml:"from"`
	To    string `yaml:"to"`
	Label string `yaml:"label,omitempty"`
}

// EntityDef is blueprint metadata for spawning world entities on first load.
type EntityDef struct {
	ID          string `yaml:"id"`
	Kind        string `yaml:"kind"`
	Name        string `yaml:"name"`
	Persistence string `yaml:"persistence"`
	TemplateID  string `yaml:"template_id,omitempty"`
	Location    string `yaml:"location"`
	Memory      string `yaml:"memory,omitempty"`
}

// WorldState is mutable session runtime (entities + node notes).
type WorldState struct {
	CurrentNodeID string            `json:"current_node_id"`
	Entities      map[string]Entity `json:"entities"`
	NodeStates    map[string]string `json:"node_states,omitempty"`
	Party         []string          `json:"party,omitempty"`
	VisitedNodes  []string          `json:"visited_nodes,omitempty"`
}

// Entity is a runtime object (NPC, artifact, ticket, …).
type Entity struct {
	ID          string `json:"id"`
	Kind        string `json:"kind"`
	Name        string `json:"name"`
	Location    string `json:"location"`
	Persistence string `json:"persistence"`
	TemplateID  string `json:"template_id,omitempty"`
	Memory      string `json:"memory,omitempty"`
	Role        string `json:"role,omitempty"`
}

// Mutation is applied when completing a node or via tools.
type Mutation struct {
	Action string `json:"action"`
	ID     string `json:"id,omitempty"`
	Key    string `json:"key,omitempty"`
	Value  string `json:"value,omitempty"`
	Note   string `json:"note,omitempty"`
}
