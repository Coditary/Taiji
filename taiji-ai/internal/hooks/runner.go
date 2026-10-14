package hooks

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// PluginFunc is an in-process hook handler registered by name.
type PluginFunc func(ctx context.Context, input ShellInput) (Result, error)

var (
	pluginMu sync.RWMutex
	plugins  = map[string]PluginFunc{}
)

// RegisterPlugin registers an in-process hook handler referenced as plugin: name.
func RegisterPlugin(name string, fn PluginFunc) {
	name = strings.TrimSpace(name)
	if name == "" || fn == nil {
		return
	}
	pluginMu.Lock()
	plugins[name] = fn
	pluginMu.Unlock()
}

// Runner executes configured hooks for agent lifecycle events.
type Runner struct {
	entries []Entry
}

// NewRunner builds a runner from hook entries.
func NewRunner(entries []Entry) *Runner {
	if len(entries) == 0 {
		return nil
	}
	cp := append([]Entry(nil), entries...)
	return &Runner{entries: cp}
}

// Enabled reports whether any hooks are configured.
func (r *Runner) Enabled() bool {
	return r != nil && len(r.entries) > 0
}

// RunPromptBefore may mutate the user prompt before the agent loop starts.
func (r *Runner) RunPromptBefore(ctx context.Context, data PromptData) (PromptData, error) {
	if !r.Enabled() {
		return data, nil
	}
	result, err := r.run(ctx, EventPromptBefore, "", ShellInput{
		Event:  EventPromptBefore,
		Env:    data.Env,
		Prompt: &data,
	})
	if err != nil {
		return data, err
	}
	if result.Block {
		return data, fmt.Errorf("%s", firstNonEmpty(result.Message, "prompt blocked by hook"))
	}
	if result.UpdatedPrompt != "" {
		data.Text = result.UpdatedPrompt
	}
	return data, nil
}

// RunContextBefore may append system context before a model call.
func (r *Runner) RunContextBefore(ctx context.Context, data ContextData) (ContextData, error) {
	if !r.Enabled() {
		return data, nil
	}
	result, err := r.run(ctx, EventContextBefore, "", ShellInput{
		Event:   EventContextBefore,
		Env:     data.Env,
		Context: &data,
	})
	if err != nil {
		return data, err
	}
	if result.AdditionalSystem != "" {
		if data.AdditionalSystem != "" {
			data.AdditionalSystem += "\n" + result.AdditionalSystem
		} else {
			data.AdditionalSystem = result.AdditionalSystem
		}
	}
	if result.AdditionalContext != "" {
		if data.AdditionalSystem != "" {
			data.AdditionalSystem += "\n" + result.AdditionalContext
		} else {
			data.AdditionalSystem = result.AdditionalContext
		}
	}
	return data, nil
}

// RunToolBefore may block or mutate tool arguments before execution.
func (r *Runner) RunToolBefore(ctx context.Context, data ToolData) (ToolData, error) {
	if !r.Enabled() {
		return data, nil
	}
	event := toolEventName(EventToolBefore, data.Tool)
	result, err := r.run(ctx, event, data.Tool, ShellInput{
		Event: event,
		Env:   data.Env,
		Tool:  &data,
	})
	if err != nil {
		return data, err
	}
	if result.Block {
		return data, fmt.Errorf("%s", firstNonEmpty(result.Message, "tool blocked by hook"))
	}
	if result.UpdatedArgs != "" {
		data.Args = result.UpdatedArgs
	}
	return data, nil
}

// RunToolAfter may mutate tool output after execution.
func (r *Runner) RunToolAfter(ctx context.Context, data ToolData) (ToolData, error) {
	if !r.Enabled() {
		return data, nil
	}
	event := toolEventName(EventToolAfter, data.Tool)
	result, err := r.run(ctx, event, data.Tool, ShellInput{
		Event: event,
		Env:   data.Env,
		Tool:  &data,
	})
	if err != nil {
		return data, err
	}
	if result.UpdatedOutput != "" {
		data.Output = result.UpdatedOutput
	} else if result.AdditionalContext != "" {
		if data.Output != "" {
			data.Output += "\n" + result.AdditionalContext
		} else {
			data.Output = result.AdditionalContext
		}
	}
	return data, nil
}

// RunFileChanged runs post-edit automation hooks.
func (r *Runner) RunFileChanged(ctx context.Context, data FileChangedData) error {
	if !r.Enabled() {
		return nil
	}
	_, err := r.run(ctx, EventFileChanged, data.Tool, ShellInput{
		Event: EventFileChanged,
		Env:   data.Env,
		File:  &data,
	})
	return err
}

// RunSession runs session lifecycle hooks (start, idle, error).
func (r *Runner) RunSession(ctx context.Context, eventName string, data SessionData) error {
	if !r.Enabled() {
		return nil
	}
	_, err := r.run(ctx, eventName, "", ShellInput{
		Event:   eventName,
		Env:     data.Env,
		Session: &data,
	})
	return err
}

func (r *Runner) run(ctx context.Context, eventName, toolName string, input ShellInput) (Result, error) {
	var merged Result
	for _, entry := range r.entries {
		if !entryMatchesEvent(entry, eventName, toolName) {
			continue
		}
		files := ExtractFilesFromArgs(toolName, toolArgs(input))
		if !conditionsMatch(entry.Conditions, toolName, files) {
			continue
		}
		if entry.Async {
			go func(entry Entry) {
				_, _ = r.runEntry(context.Background(), entry, input)
			}(entry)
			continue
		}
		result, err := r.runEntry(ctx, entry, input)
		if err != nil {
			return merged, err
		}
		merged = merged.mergeInto(result)
		if merged.Block {
			return merged, nil
		}
	}
	return merged, nil
}

func (r *Runner) runEntry(ctx context.Context, entry Entry, input ShellInput) (Result, error) {
	var merged Result
	for _, action := range entry.Actions {
		result, err := r.runAction(ctx, input.Env.WorkDir, action, input)
		if err != nil {
			return merged, err
		}
		merged = merged.mergeInto(result)
		if merged.Block {
			return merged, nil
		}
	}
	return merged, nil
}

func (r *Runner) runAction(ctx context.Context, workDir string, spec ActionSpec, input ShellInput) (Result, error) {
	if name := strings.TrimSpace(spec.Plugin); name != "" {
		pluginMu.RLock()
		fn := plugins[name]
		pluginMu.RUnlock()
		if fn == nil {
			return Result{}, fmt.Errorf("unknown hook plugin %q", name)
		}
		return fn(ctx, input)
	}
	if strings.TrimSpace(spec.Bash) != "" || strings.TrimSpace(spec.Command) != "" {
		return runShellAction(ctx, workDir, spec, input)
	}
	return Result{}, nil
}

func toolArgs(input ShellInput) string {
	if input.Tool == nil {
		return ""
	}
	return input.Tool.Args
}
