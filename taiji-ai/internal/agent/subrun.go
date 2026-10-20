package agent

// SubRunConfig customizes isolated short agent runs (e.g. delegate_task).
type SubRunConfig struct {
	Enabled          bool
	MaxSteps         int
	SkipTurnRouter   bool
	SkipHooks        bool
	SkipMemoryPasses bool
	SystemPrompt     string
}
