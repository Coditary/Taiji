package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
)

const (
	envTUIStartProject    = "TAIJI_START_PROJECT"
	envTUIStartNewSession = "TAIJI_START_NEW_SESSION"
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
	remaining, projectDir, newSession := consumeTUIProjectArg(extraArgs)
	applyTUIStartupEnv(projectDir, newSession)
	argv := append([]string{bin}, remaining...)
	return syscall.Exec(bin, argv, os.Environ())
}

func applyTUIStartupEnv(projectDir string, newSession bool) {
	if projectDir != "" {
		_ = os.Setenv(envTUIStartProject, projectDir)
	}
	if newSession {
		_ = os.Setenv(envTUIStartNewSession, "1")
	}
}

// tryLaunchTUIProjectShortcut handles `taiji .` before Cobra treats "." as a subcommand name.
func tryLaunchTUIProjectShortcut(argv []string) (bool, error) {
	pathArgs := projectShortcutPositionalArgs(argv)
	if len(pathArgs) != 1 {
		return false, nil
	}
	_, projectDir, newSession := consumeTUIProjectArg(pathArgs)
	if !newSession {
		return false, nil
	}
	bin, err := resolveTUIExecutable()
	if err != nil {
		return true, err
	}
	applyTUIStartupEnv(projectDir, true)
	return true, syscall.Exec(bin, []string{bin}, os.Environ())
}

var cliSubcommandNames = map[string]struct{}{
	"tui":         {},
	"theme":       {},
	"run":         {},
	"chat":        {},
	"serve":       {},
	"providers":   {},
	"sessions":    {},
	"rag":         {},
	"import-card": {},
}

func projectShortcutPositionalArgs(argv []string) []string {
	if len(argv) == 0 {
		return nil
	}
	var positional []string
	for _, arg := range argv {
		if arg == "--local" || arg == "--local-only" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		positional = append(positional, arg)
	}
	if len(positional) == 0 {
		return nil
	}
	if _, known := cliSubcommandNames[positional[0]]; known {
		return nil
	}
	return positional
}

// consumeTUIProjectArg recognizes `taiji .` (or `taiji <dir>`) as "open a new session in this folder".
func consumeTUIProjectArg(args []string) (remaining []string, projectDir string, newSession bool) {
	if len(args) != 1 {
		return args, "", false
	}
	arg := strings.TrimSpace(args[0])
	if arg == "" || strings.HasPrefix(arg, "-") {
		return args, "", false
	}
	if arg != "." && arg != "./" {
		info, err := os.Stat(arg)
		if err != nil || !info.IsDir() {
			return args, "", false
		}
	}
	abs, err := filepath.Abs(arg)
	if err != nil {
		return args, "", false
	}
	return nil, abs, true
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
