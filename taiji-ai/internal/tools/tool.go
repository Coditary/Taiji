package tools

import "context"

// Tool is a callable agent tool.
type Tool interface {
	Name() string
	Description() string
	Parameters() map[string]any
	Run(ctx context.Context, args map[string]any) (string, error)
}

// CatalogHidden marks tools that execute at runtime but are omitted from LLM tool catalogs.
type CatalogHidden interface {
	HideFromCatalog() bool
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
