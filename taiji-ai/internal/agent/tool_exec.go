package agent

import (
	"context"
	"sync"
	"time"

	"github.com/coditary/taiji/internal/hooks"
	"github.com/coditary/taiji/internal/question"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/tools"
)

type toolExecOutcome struct {
	result     tools.RunResult
	err        error
	durationMs int64
	duplicate  bool
	text       string
}

func executeToolCalls(ctx context.Context, hookRunner *hooks.Runner, reg *tools.Registry, env hooks.Env, calls []provider.ToolCall, messages []provider.Message) []toolExecOutcome {
	outcomes := make([]toolExecOutcome, len(calls))
	var wg sync.WaitGroup
	for i, tc := range calls {
		if isDuplicateToolCall(messages, tc) {
			outcomes[i] = toolExecOutcome{
				duplicate: true,
				text:      duplicateToolObservation(messages, tc),
			}
			continue
		}
		wg.Add(1)
		go func(i int, tc provider.ToolCall) {
			defer wg.Done()
			outcomes[i] = runToolCall(ctx, hookRunner, reg, env, tc)
		}(i, tc)
	}
	wg.Wait()
	return outcomes
}

func runToolCall(ctx context.Context, hookRunner *hooks.Runner, reg *tools.Registry, env hooks.Env, tc provider.ToolCall) toolExecOutcome {
	toolData := hooks.ToolData{
		Env:    env,
		Tool:   tc.Name,
		ToolID: tc.ID,
		Args:   tc.Args,
	}
	if hookRunner != nil {
		var err error
		toolData, err = hookRunner.RunToolBefore(ctx, toolData)
		if err != nil {
			return toolExecOutcome{err: err, durationMs: 0}
		}
	}

	start := time.Now()
	result, err := reg.Run(question.WithToolCall(ctx, tc.ID), toolData.Tool, toolData.Args)
	outcome := toolExecOutcome{
		result:     result,
		err:        err,
		durationMs: time.Since(start).Milliseconds(),
	}
	if err != nil {
		return outcome
	}

	toolData.Output = result.Text
	if hookRunner != nil {
		var hookErr error
		toolData, hookErr = hookRunner.RunToolAfter(ctx, toolData)
		if hookErr != nil {
			outcome.err = hookErr
			return outcome
		}
		outcome.result.Text = toolData.Output

		if files := hooks.ExtractFilesFromArgs(toolData.Tool, toolData.Args); len(files) > 0 {
			_ = hookRunner.RunFileChanged(ctx, hooks.FileChangedData{
				Env:   env,
				Tool:  toolData.Tool,
				Files: files,
			})
		}
	}
	return outcome
}
