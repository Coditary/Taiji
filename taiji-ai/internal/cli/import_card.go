package cli

import (
	"fmt"

	"github.com/coditary/taiji/internal/session"
	"github.com/spf13/cobra"
)

func newImportCardCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import-card [file.json|file.png]",
		Short: "Import a SillyTavern chara_card_v2 into a markdown chat template",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Config == nil {
				return fmt.Errorf("config not loaded")
			}
			path, lore, err := session.ImportCharaCard(app.Config.Root, args[0], "")
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote template: %s\n", path)
			if lore != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Lore pack: %s (under .taiji/packs/)\n", lore)
			}
			return nil
		},
	}
	return cmd
}
