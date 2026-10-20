package runtime

// RegisterExtendedTools registers a hook invoked at the end of New (e.g. delegate_task).
var RegisterExtendedTools func(*Runtime)

func runExtendedTools(rt *Runtime) {
	if RegisterExtendedTools != nil && rt != nil {
		RegisterExtendedTools(rt)
	}
}
