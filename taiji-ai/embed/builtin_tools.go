package main

import (
	"sort"
	"strings"

	"github.com/coditary/taiji/internal/tools"
)

func listBuiltinToolNames(reg *tools.Registry, mcpTools []mcpToolPayload) []string {
	if reg == nil {
		return nil
	}
	mcpNames := map[string]struct{}{}
	for _, t := range mcpTools {
		if name := strings.TrimSpace(t.RegisteredName); name != "" {
			mcpNames[name] = struct{}{}
		}
		if name := strings.TrimSpace(t.Name); name != "" {
			mcpNames[name] = struct{}{}
		}
	}
	var names []string
	for _, d := range reg.Definitions() {
		name := strings.TrimSpace(d.Name)
		if name == "" {
			continue
		}
		if _, ok := mcpNames[name]; ok {
			continue
		}
		if _, _, ok := tools.SplitNamespaced(name); ok {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
