package builtin

import (
	"testing"

	"github.com/coditary/taiji/internal/tools"
)

func TestRegisterOmitsBatchEditFromCatalog(t *testing.T) {
	reg := tools.NewRegistry()
	Register(reg)
	for _, d := range reg.Definitions() {
		if d.Name == "batch_edit" {
			t.Fatal("batch_edit should be hidden from LLM catalog")
		}
	}
	got := false
	for _, d := range reg.Definitions() {
		if d.Name == "edit_file" {
			got = true
		}
	}
	if !got {
		t.Fatal("edit_file missing from catalog")
	}
}
