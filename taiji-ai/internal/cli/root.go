package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// App holds CLI state.
type App struct {
	Config *taijicfg.Config
	Root   *cobra.Command
}

// New creates the root CLI.
func New() (*App, error) {
	app := &App{}
	root := &cobra.Command{
		Use:   "taiji",
		Short: "Agent orchestrator for tool-using AI loops",
		Long: `Taiji runs agent loops with YAML-defined providers, built-in tools, MCP, and skills.

Providers are configured in .taiji/config.yaml — no code changes required to add backends.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			local, err := cmd.Root().PersistentFlags().GetBool("local")
			if err != nil {
				return err
			}
			localOnly, err := cmd.Root().PersistentFlags().GetBool("local-only")
			if err != nil {
				return err
			}
			if localOnly {
				local = true
			}
			cfg, err := taijicfg.Load(taijicfg.LoadOptions{Local: local})
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			app.Config = cfg
			return nil
		},
	}
	root.PersistentFlags().Bool(
		"local",
		false,
		"project root is the current directory; RAG writes stay local, reads merge local+global",
	)
	root.PersistentFlags().Bool(
		"local-only",
		false,
		"like --local but RAG reads only the local store (no global knowledge)",
	)
	root.AddCommand(
		newRunCmd(app),
		newChatCmd(app),
		newServeCmd(app),
		newProvidersCmd(app),
		newSessionsCmd(app),
		newRagCmd(app),
	)
	app.Root = root
	return app, nil
}

// Execute runs the CLI.
func (a *App) Execute() {
	if err := a.Root.Execute(); err != nil {
		fmt.Fprintln(a.Root.ErrOrStderr(), err)
		os.Exit(1)
	}
}
