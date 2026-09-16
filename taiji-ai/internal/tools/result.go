package tools

import "context"

// RunResult is the output of a tool execution.
type RunResult struct {
	Text string
	Data map[string]any
}

// MetadataTool optionally returns structured metadata alongside text output.
type MetadataTool interface {
	Tool
	RunWithMetadata(ctx context.Context, args map[string]any) (RunResult, error)
}
