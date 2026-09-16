package cli

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/coditary/taiji/internal/session"
)

func newSessionsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sessions",
		Short: "List and inspect saved chat sessions",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List saved sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			summaries, err := session.ListAll(app.Config.Root)
			if err != nil {
				return err
			}
			if len(summaries) == 0 {
				fmt.Fprintln(os.Stdout, "(no sessions)")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tPROVIDER\tMSGS\tCREATED")
			for _, s := range summaries {
				name := s.Name
				if name == "" {
					name = "-"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
					s.ID,
					name,
					orDash(s.Provider),
					s.Messages,
					s.CreatedAt.Format("2006-01-02 15:04"),
				)
			}
			return w.Flush()
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <session-id>",
		Short: "Delete a saved session",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(args[0])
			if err := session.Delete(app.Config.Root, id); err != nil {
				return err
			}
			fmt.Fprintf(os.Stdout, "deleted session %q\n", id)
			return nil
		},
	})

	return cmd
}

func orDash(v string) string {
	if v == "" {
		return "-"
	}
	return v
}
