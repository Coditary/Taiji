package builtin

import "context"

type Grep struct{}

func (Grep) Name() string { return "grep" }

func (Grep) Description() string {
	return "Search file contents. Uses ripgrep (rg) when installed; otherwise a built-in search. Supports glob, output_mode (content, files_with_matches, count), context_lines, and type when rg is available."
}

func (Grep) Parameters() map[string]any {
	return contentSearchToolParameters()
}

func (Grep) Run(ctx context.Context, args map[string]any) (string, error) {
	return RunContentSearch(ctx, args)
}
