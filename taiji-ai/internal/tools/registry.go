package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// Registry holds tools by name.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates an empty registry.
func NewRegistry() *Registry {
	return &Registry{tools: map[string]Tool{}}
}

// Register adds a tool.
func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

// Get returns the tool registered under name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Definitions returns provider tool definitions.
func (r *Registry) Definitions() []Definition {
	tools := r.List()
	out := make([]Definition, 0, len(tools))
	for _, t := range tools {
		out = append(out, ToDefinition(t))
	}
	return out
}

// Run executes a tool by name with JSON args.
func (r *Registry) Run(ctx context.Context, name string, argsJSON string) (RunResult, error) {
	r.mu.RLock()
	t, ok := r.tools[name]
	r.mu.RUnlock()
	if !ok {
		return RunResult{}, fmt.Errorf("unknown tool %q", name)
	}
	args := map[string]any{}
	if stringsTrim(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return RunResult{}, fmt.Errorf("parse tool args: %w", err)
		}
	}
	if mt, ok := t.(MetadataTool); ok {
		return mt.RunWithMetadata(ctx, args)
	}
	out, err := t.Run(ctx, args)
	return RunResult{Text: out}, err
}

func stringsTrim(s string) string {
	i := 0
	j := len(s)
	for i < j && (s[i] == ' ' || s[i] == '\n' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\n' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
