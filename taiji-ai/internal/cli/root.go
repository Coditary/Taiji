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

Project settings live under XDG data dirs by default; use --local for a portable .taiji folder in the project.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI(args)
		},
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
			if local {
				_ = os.Setenv("TAIJI_LOCAL", "1")
			} else {
				_ = os.Unsetenv("TAIJI_LOCAL")
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
		newTuiCmd(),
		newThemeCmd(),
		newRunCmd(app),
		newChatCmd(app),
		newServeCmd(app),
		newProvidersCmd(app),
		newSessionsCmd(app),
		newRagCmd(app),
		newImportCardCmd(app),
	)
	app.Root = root
	return app, nil
}

// Execute runs the CLI.
func (a *App) Execute() {
	launched, err := tryLaunchTUIProjectShortcut(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if launched {
		return
	}
	if err := a.Root.Execute(); err != nil {
		fmt.Fprintln(a.Root.ErrOrStderr(), err)
		os.Exit(1)
	}
}
