package tools

import (
	"context"
	"strings"
)

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
	Namespace   string
	Description string
	Parameters  map[string]any
}

// ToDefinition converts a Tool to a provider definition.
func ToDefinition(t Tool) Definition {
	def := Definition{
		Name:        t.Name(),
		Namespace:   NamespaceBuiltin,
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
	if p, ok := t.(NamespaceProvider); ok {
		if ns := strings.TrimSpace(p.ToolNamespace()); ns != "" {
			def.Namespace = ns
		}
	}
	return def
}

