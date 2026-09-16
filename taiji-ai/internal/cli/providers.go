package cli

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func newProvidersCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "List, refresh, and select LLM providers",
		Long: `Providers are saved in .taiji/config.yaml.

Refresh the catalog from a running Wuji daemon:
  taiji providers refresh

Pick the default provider for taiji run:
  taiji providers use vllm

With wuji.auto_sync: true, refresh runs automatically on each command.`,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List saved providers",
			RunE: func(cmd *cobra.Command, args []string) error {
				return printProviderTable(app.Config)
			},
		},
		&cobra.Command{
			Use:     "refresh",
			Aliases: []string{"sync"},
			Short:   "Refresh provider catalog from Wuji and save to .taiji/config.yaml",
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx := cmd.Context()
				if ctx == nil {
					ctx = context.Background()
				}
				res, err := provider.RefreshProvidersFromWuji(ctx, app.Config)
				if err != nil {
					return err
				}
				if res.Connected {
					if err := app.Config.Save(); err != nil {
						return err
					}
					res.Saved = true
				}
				fmt.Fprintf(os.Stdout, "connected: %v\n", res.Connected)
				fmt.Fprintf(os.Stdout, "endpoint: %s\n", res.Endpoint)
				fmt.Fprintf(os.Stdout, "added: %d\n", res.Added)
				fmt.Fprintf(os.Stdout, "updated: %d\n", res.Updated)
				fmt.Fprintf(os.Stdout, "saved: %v\n", res.Saved)
				fmt.Fprintf(os.Stdout, "message: %s\n", res.Message)
				if res.Connected {
					return printProviderTable(app.Config)
				}
				return nil
			},
		},
		&cobra.Command{
			Use:   "use <provider-id>",
			Short: "Set the default provider and save",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				id := strings.TrimSpace(args[0])
				if id == "" {
					return fmt.Errorf("provider id is required")
				}
				if app.Config.Providers == nil || app.Config.Providers[id].Type == "" {
					return fmt.Errorf("provider %q not found — run: taiji providers refresh", id)
				}
				app.Config.DefaultProvider = id
				if err := app.Config.Save(); err != nil {
					return err
				}
				fmt.Fprintf(os.Stdout, "default provider set to %q\n", id)
				return nil
			},
		},
	)

	return cmd
}

func printProviderTable(cfg *taijicfg.Config) error {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tDRIVER\tMODEL\tFLAGS")
	ids := make([]string, 0, len(cfg.Providers))
	for id := range cfg.Providers {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		pc := cfg.Providers[id]
		flags := providerFlags(pc, id == cfg.DefaultProvider)
		driverID := strings.TrimSpace(pc.ProviderID)
		if driverID == "" {
			driverID = "-"
		}
		model := strings.TrimSpace(pc.Model)
		if model == "" {
			model = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", id, pc.Type, driverID, model, flags)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	if cfg.DefaultProvider != "" {
		fmt.Fprintf(os.Stderr, "\ndefault: %s\n", cfg.DefaultProvider)
	} else {
		fmt.Fprintln(os.Stderr, "\ndefault: (not set — run: taiji providers use <id>)")
	}
	return nil
}

func providerFlags(pc taijicfg.ProviderConfig, isDefault bool) string {
	var flags []string
	if isDefault {
		flags = append(flags, "default")
	}
	if pc.Pinned {
		flags = append(flags, "pinned")
	}
	if pc.Synced {
		flags = append(flags, "synced")
	}
	if len(flags) == 0 {
		return "-"
	}
	return strings.Join(flags, ",")
}
