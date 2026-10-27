package tools

import "strings"

// FilterRegistry returns a registry containing only the named tools from main.
func FilterRegistry(main *Registry, allow []string) *Registry {
	sub := NewRegistry()
	if main == nil {
		return sub
	}
	for _, name := range allow {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if tool, ok := main.Get(name); ok {
			sub.Register(tool)
		}
	}
	return sub
}
