package builtin

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type RunTests struct{}

func (RunTests) Name() string { return "run_tests" }

func (RunTests) Description() string {
	return "Run Go tests with compact JSON output: one-line summary on success, failed tests and logs only on failure."
}

func (RunTests) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pkg": map[string]any{
				"type":        "string",
				"description": "Package pattern (default ./...)",
			},
			"run": map[string]any{
				"type":        "string",
				"description": "Optional -run regex",
			},
			"race": map[string]any{
				"type":        "boolean",
				"description": "Enable -race",
			},
			"timeout": map[string]any{
				"type":        "string",
				"description": "Overall timeout (default 30s)",
			},
		},
	}
}

func (RunTests) Run(ctx context.Context, args map[string]any) (string, error) {
	pkg, _ := args["pkg"].(string)
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		pkg = "./..."
	}
	runPat, _ := args["run"].(string)
	runPat = strings.TrimSpace(runPat)
	race, _ := args["race"].(bool)
	timeoutRaw, _ := args["timeout"].(string)
	timeout, err := parseTestTimeout(timeoutRaw)
	if err != nil {
		return "", err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	goArgs := []string{"test", "-json", pkg}
	if runPat != "" {
		goArgs = append(goArgs, "-run", runPat)
	}
	if race {
		goArgs = append(goArgs, "-race")
	}
	cmd := exec.CommandContext(runCtx, "go", goArgs...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	cmdErr := cmd.Run()

	parsed, parseErr := parseGoTestJSON(&buf)
	if parseErr != nil {
		return "", parseErr
	}
	out := formatGoTestSummary(parsed)
	if cmdErr != nil && parsed.Passed && parsed.TestCount == 0 && parsed.PackageCount == 0 {
		tail := strings.TrimSpace(buf.String())
		if tail == "" {
			return "", cmdErr
		}
		return tail, cmdErr
	}
	if !parsed.Passed {
		return out, fmt.Errorf("tests failed")
	}
	if cmdErr != nil {
		return out, cmdErr
	}
	return out, nil
}
