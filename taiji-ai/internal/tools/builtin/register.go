package builtin

import "github.com/coditary/taiji/internal/tools"

// Register adds built-in tools to the registry.
func Register(r *tools.Registry) {
	r.Register(ListDir{})
	r.Register(GlobSearch{})
	r.Register(Grep{})
	r.Register(ReadFile{})
	r.Register(WriteFile{})
	r.Register(Shell{})
}
