package agent

import (
	"strings"

	"github.com/coditary/taiji/internal/runtime"
)

// WireRuntimeTools registers agent-level tools on a constructed runtime.
func WireRuntimeTools(rt *runtime.Runtime) {
	if rt == nil || rt.Tools == nil || rt.Providers == nil {
		return
	}
	workDir := "."
	if rt.Config != nil && strings.TrimSpace(rt.Config.Root) != "" {
		workDir = rt.Config.Root
	}
	EnableDelegateTask(rt.Tools, DelegateTaskConfig{
		Providers: rt.Providers,
		MainTools: rt.Tools,
		Config:    rt.Config,
		WorkDir:   workDir,
	})
}
