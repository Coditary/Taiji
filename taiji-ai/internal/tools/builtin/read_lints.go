package builtin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const lintCommandTimeout = 45 * time.Second

type ReadLints struct{}

func (ReadLints) Name() string { return "read_lints" }

func (ReadLints) Description() string {
	return "Run workspace linters for paths (files or directories). Supports Go (go vet / golangci-lint) and JS/TS (npm run lint / eslint when configured)."
}

func (ReadLints) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"paths": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Files or directories (default: project root)",
			},
		},
	}
}

func (ReadLints) Run(ctx context.Context, args map[string]any) (string, error) {
	paths := stringSliceArg(args["paths"])
	if len(paths) == 0 {
		root := workspaceProjectRoot()
		if root == "" {
			root = "."
		}
		paths = []string{root}
	}
	var sections []string
	for _, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		abs, err := explorePath(p)
		if err != nil {
			return "", err
		}
		sec, err := lintAtPath(ctx, abs)
		if err != nil {
			sections = append(sections, fmt.Sprintf("## %s\nerror: %v", abs, err))
			continue
		}
		if strings.TrimSpace(sec) == "" {
			sec = "(no issues reported)"
		}
		sections = append(sections, fmt.Sprintf("## %s\n%s", abs, sec))
	}
	if len(sections) == 0 {
		return "", fmt.Errorf("no paths to lint")
	}
	return strings.Join(sections, "\n\n"), nil
}

func lintAtPath(ctx context.Context, abs string) (string, error) {
	info, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		if hasFile(filepath.Join(abs, "go.mod")) {
			return runLintCommand(ctx, abs, goLintCommands(abs))
		}
		if hasFile(filepath.Join(abs, "package.json")) {
			return runLintCommand(ctx, abs, jsLintCommands(abs))
		}
		return "(no known linter for directory)", nil
	}
	switch strings.ToLower(filepath.Ext(abs)) {
	case ".go":
		return runLintCommand(ctx, filepath.Dir(abs), goLintCommands(filepath.Dir(abs)))
	default:
		dir := filepath.Dir(abs)
		if hasFile(filepath.Join(dir, "package.json")) {
			return runLintCommand(ctx, dir, jsLintCommands(dir))
		}
		return "(no linter for file type)", nil
	}
}

func goLintCommands(dir string) []lintCmd {
	var cmds []lintCmd
	if _, err := exec.LookPath("golangci-lint"); err == nil {
		cmds = append(cmds, lintCmd{"golangci-lint", []string{"run", "./..."}, dir})
	} else if toolsSettings.AutoVetOrDefault() {
		cmds = append(cmds, lintCmd{"go", []string{"vet", "./..."}, dir})
	}
	return cmds
}

func jsLintCommands(dir string) []lintCmd {
	var cmds []lintCmd
	if hasFile(filepath.Join(dir, "package.json")) {
		cmds = append(cmds, lintCmd{"npm", []string{"run", "lint", "--if-present"}, dir})
	}
	if _, err := exec.LookPath("npx"); err == nil {
		if hasFile(filepath.Join(dir, ".eslintrc")) || hasFile(filepath.Join(dir, ".eslintrc.json")) || hasFile(filepath.Join(dir, "eslint.config.js")) {
			cmds = append(cmds, lintCmd{"npx", []string{"eslint", "."}, dir})
		}
	}
	return cmds
}

type lintCmd struct {
	bin  string
	args []string
	dir  string
}

func runLintCommand(ctx context.Context, dir string, cmds []lintCmd) (string, error) {
	if len(cmds) == 0 {
		return "(no linter available)", nil
	}
	var parts []string
	for _, c := range cmds {
		out, err := runLintCmd(ctx, c)
		label := c.bin + " " + strings.Join(c.args, " ")
		if err != nil {
			parts = append(parts, fmt.Sprintf("$ %s\n%s", label, out))
			continue
		}
		if strings.TrimSpace(out) == "" {
			parts = append(parts, fmt.Sprintf("$ %s\n(clean)", label))
		} else {
			parts = append(parts, fmt.Sprintf("$ %s\n%s", label, out))
		}
	}
	return strings.Join(parts, "\n\n"), nil
}

func runLintCmd(ctx context.Context, c lintCmd) (string, error) {
	runCtx, cancel := context.WithTimeout(ctx, lintCommandTimeout)
	defer cancel()
	cmd := exec.CommandContext(runCtx, c.bin, c.args...)
	cmd.Dir = c.dir
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil && text == "" {
		text = err.Error()
	}
	return text, err
}

func hasFile(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
