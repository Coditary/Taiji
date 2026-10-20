package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/ui"
)

func newThemeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "theme",
		Short: "Taiji TUI color themes (.theme stylesheets)",
	}
	cmd.AddCommand(newThemeInitCmd(), newThemeListCmd())
	return cmd
}

func newThemeInitCmd() *cobra.Command {
	var name string
	var output string
	var force bool
	var printStdout bool

	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Write a commented starter theme to ~/.config/taiji/themes/",
		Long: `Creates a self-documenting .theme file you can edit and select in the TUI.

The theme id is the filename without .theme (letters, digits, underscore, hyphen).
Pick it under Settings → Appearance → Color theme, or set ui.theme in .taiji/config.yaml.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := strings.TrimSpace(name)
			if len(args) > 0 {
				id = strings.TrimSpace(args[0])
			}
			if id == "" {
				id = "custom"
			}
			if !taijicfg.ValidThemeID(id) {
				return fmt.Errorf("invalid theme name %q: use letters, digits, underscore, or hyphen", id)
			}

			body := ui.ThemeTemplate(id)
			if printStdout {
				fmt.Fprint(cmd.OutOrStdout(), body)
				if !strings.HasSuffix(body, "\n") {
					fmt.Fprintln(cmd.OutOrStdout())
				}
				return nil
			}

			dest := strings.TrimSpace(output)
			if dest == "" {
				dir := taijicfg.GlobalUIThemesDir()
				if dir == "" {
					return fmt.Errorf("could not resolve config directory (HOME unset?)")
				}
				if err := os.MkdirAll(dir, 0o755); err != nil {
					return fmt.Errorf("create themes dir: %w", err)
				}
				dest = filepath.Join(dir, id+".theme")
			} else {
				dest = filepath.Clean(dest)
				if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
					return fmt.Errorf("create output directory: %w", err)
				}
			}

			if !force {
				if _, err := os.Stat(dest); err == nil {
					return fmt.Errorf("%s already exists (use --force to overwrite)", dest)
				}
			}
			if err := os.WriteFile(dest, []byte(body), 0o644); err != nil {
				return fmt.Errorf("write theme: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", dest)
			fmt.Fprintf(cmd.OutOrStdout(), "Select theme id \"%s\" in the TUI settings or set ui.theme in config.\n", id)
			return nil
		},
	}
	cmd.Flags().StringVarP(&name, "name", "n", "custom", "theme id / filename stem")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write to this path instead of the config themes dir")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing file")
	cmd.Flags().BoolVar(&printStdout, "print", false, "print template to stdout instead of writing a file")
	return cmd
}

func newThemeListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List built-in and user theme ids",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Built-in (embedded in taiji-tui):")
			for _, id := range []string{
				"dark",
				"high_contrast",
				"gruvbox_dark",
				"gruvbox_light",
				"catppuccin_mocha",
				"catppuccin_latte",
				"nord",
				"dracula",
				"rose_pine",
				"solarized_dark",
				"everforest_dark",
				"one_dark",
			} {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", id)
			}
			dir := taijicfg.GlobalUIThemesDir()
			if dir == "" {
				return nil
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				if os.IsNotExist(err) {
					fmt.Fprintf(cmd.OutOrStdout(), "\nUser themes (%s): (directory not created yet — run: taiji theme init)\n", dir)
					return nil
				}
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\nUser themes (%s):\n", dir)
			found := false
			for _, ent := range entries {
				if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".theme") {
					continue
				}
				id := strings.TrimSuffix(ent.Name(), ".theme")
				if !taijicfg.ValidThemeID(id) {
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", id)
				found = true
			}
			if !found {
				fmt.Fprintln(cmd.OutOrStdout(), "  (none — run: taiji theme init)")
			}
			return nil
		},
	}
}
