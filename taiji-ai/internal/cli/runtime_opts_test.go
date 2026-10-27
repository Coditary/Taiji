package cli

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestRuntimeOptionsLocalOnlyImpliesLocal(t *testing.T) {
	app, err := New()
	if err != nil {
		t.Fatal(err)
	}
	_ = app.Root.PersistentFlags().Set("local-only", "true")
	sub := &cobra.Command{Use: "chat"}
	app.Root.AddCommand(sub)
	opts := app.runtimeOptions(sub)
	if !opts.Local || !opts.LocalOnly {
		t.Fatalf("opts=%+v", opts)
	}
}
