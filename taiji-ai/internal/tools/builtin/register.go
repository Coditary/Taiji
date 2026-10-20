package builtin

import "github.com/coditary/taiji/internal/tools"

// Register adds built-in tools to the registry.
func Register(r *tools.Registry) {
	r.Register(ListDir{})
	r.Register(GlobSearch{})
	r.Register(Grep{})
	r.Register(ReadFile{})
	r.Register(WriteFile{})
	r.Register(EditFile{})
	r.Register(BatchEdit{})
	r.Register(ManageFile{})
	r.Register(GoSymbolIntel{})
	r.Register(FileOutline{})
	r.Register(BgProcess{})
	r.Register(WorkspaceCheckpoint{})
	r.Register(TodoPlanTool{})
	r.Register(Shell{})
	r.Register(RunTests{})
	r.Register(WebFetch{})
	r.Register(GitInspect{})
}
