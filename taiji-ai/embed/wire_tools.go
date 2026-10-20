package embed

import (
	"github.com/coditary/taiji/internal/agent"
	"github.com/coditary/taiji/internal/runtime"
)

func init() {
	runtime.RegisterExtendedTools = agent.WireRuntimeTools
}
