package tools

import "context"

// Tool is a callable agent tool.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Run(ctx context.Context, args map[string]any) (string, error)
}

// Definition describes a tool for LLM providers.
type Definition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

// ToDefinition converts a Tool to a provider definition.
func ToDefinition(t Tool) Definition {
	return Definition{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}
