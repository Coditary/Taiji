package cli

import (
	"github.com/coditary/taiji/internal/runtime"
	"github.com/spf13/cobra"
)

func (a *App) runtimeOptions(cmd *cobra.Command) runtime.Options {
	local, _ := cmd.Root().PersistentFlags().GetBool("local")
	localOnly, _ := cmd.Root().PersistentFlags().GetBool("local-only")
	if localOnly {
		local = true
	}
	return runtime.Options{Local: local, LocalOnly: localOnly}
}
