package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
)

func newTuiCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "tui",
		Short: "Start the terminal UI",
		RunE: func(cmd *cobra.Command, args []string) error {
			return launchTUI(args)
		},
	}
}

func launchTUI(extraArgs []string) error {
	bin, err := resolveTUIExecutable()
	if err != nil {
		return err
	}
	argv := append([]string{bin}, extraArgs...)
	return syscall.Exec(bin, argv, os.Environ())
}

func resolveTUIExecutable() (string, error) {
	if self, err := os.Executable(); err == nil {
		sibling := filepath.Join(filepath.Dir(self), "taiji-tui")
		if info, err := os.Stat(sibling); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return sibling, nil
		}
	}
	path, err := exec.LookPath("taiji-tui")
	if err != nil {
		return "", fmt.Errorf("taiji-tui not found: install taiji-tui next to taiji or add it to PATH")
	}
	return path, nil
}
