package builtin

import (
	"context"
	"fmt"

	"github.com/coditary/taiji/internal/packs"
	"github.com/coditary/taiji/internal/tools"
)

// RememberInPack appends facts to a context pack node (restricted to pack dirs).
type RememberInPack struct {
	PackDirs []string
	OnReload func() error
}

func (t RememberInPack) Name() string { return "remember_in_pack" }

func (t RememberInPack) Description() string {
	return "Append a markdown fact to a context pack node under .taiji/packs (safe write; no arbitrary paths)."
}

func (t RememberInPack) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pack_name": map[string]any{"type": "string", "description": "Pack folder name"},
			"node_name": map[string]any{"type": "string", "description": "Relative markdown path (e.g. lore/tavern)"},
			"content":   map[string]any{"type": "string", "description": "Markdown to append"},
		},
		"required": []string{"pack_name", "node_name", "content"},
	}
}

func (t RememberInPack) Run(ctx context.Context, args map[string]any) (string, error) {
	path, err := packs.AppendToNode(t.PackDirs, stringArg(args, "pack_name"), stringArg(args, "node_name"), stringArg(args, "content"))
	if err != nil {
		return "", err
	}
	if t.OnReload != nil {
		if err := t.OnReload(); err != nil {
			return "", fmt.Errorf("pack reindex: %w", err)
		}
	}
	return fmt.Sprintf("appended to %s", path), nil
}

// EnableActorTools registers actor state tooling.
func EnableActorTools(reg *tools.Registry) {
	if reg == nil {
		return
	}
	reg.Register(UpdateActorState{})
}

// EnablePackRemember registers remember_in_pack when pack dirs are configured.
func EnablePackRemember(reg *tools.Registry, packDirs []string, reload func() error) {
	if reg == nil || len(packDirs) == 0 {
		return
	}
	reg.Register(RememberInPack{PackDirs: packDirs, OnReload: reload})
}
