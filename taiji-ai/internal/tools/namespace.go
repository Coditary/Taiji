package tools

import "strings"

// NamespaceBuiltin is the namespace for first-party Taiji built-in tools.
const NamespaceBuiltin = "taiji"

// NamespaceMCPMeta is the namespace for Taiji tools that operate across MCP servers.
const NamespaceMCPMeta = "mcp"

const namespaceSep = "__"

// NamespaceProvider reports a tool's namespace for catalogs and provider metadata.
type NamespaceProvider interface {
	ToolNamespace() string
}

// Namespaced builds a stable registered name: "<namespace>__<local>".
// Segments are trimmed; embedded separators in namespace or local are collapsed to "_".
func Namespaced(namespace, local string) string {
	ns := sanitizeNamespaceSegment(namespace)
	loc := sanitizeNamespaceSegment(local)
	if ns == "" {
		return loc
	}
	if loc == "" {
		return ns
	}
	return ns + namespaceSep + loc
}

// SplitNamespaced splits a registered namespaced tool name.
func SplitNamespaced(registered string) (namespace, local string, ok bool) {
	registered = strings.TrimSpace(registered)
	i := strings.Index(registered, namespaceSep)
	if i <= 0 || i+len(namespaceSep) >= len(registered) {
		return "", registered, false
	}
	return registered[:i], registered[i+len(namespaceSep):], true
}

func sanitizeNamespaceSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, namespaceSep, "_")
	return s
}
