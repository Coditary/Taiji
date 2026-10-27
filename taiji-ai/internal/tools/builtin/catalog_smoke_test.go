package builtin

import (
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

func TestRegisteredToolsExposeMetadata(t *testing.T) {
	reg := tools.NewRegistry()
	Register(reg)
	for _, tool := range reg.List() {
		if tool.Name() == "" {
			t.Fatalf("%T missing name", tool)
		}
		if tool.Description() == "" {
			t.Fatalf("%s missing description", tool.Name())
		}
		if tool.Parameters() == nil {
			t.Fatalf("%s missing parameters", tool.Name())
		}
	}
}
