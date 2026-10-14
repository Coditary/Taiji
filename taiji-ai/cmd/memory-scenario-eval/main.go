// memory-scenario-eval runs deterministic memory store/recall scenario tests (no live LLM).
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

const targetScore = 90.0

func main() {
	root, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "cwd: %v\n", err)
		os.Exit(1)
	}
	if v := os.Getenv("TAIJI_EVAL_ROOT"); v != "" {
		root = v
	}

	cmd := exec.Command("go", "test", "./internal/memory/", "-run", "TestMemoryScenarioEval", "-count=1", "-v")
	cmd.Dir = filepath.Join(root, "taiji-ai")
	if _, err := os.Stat(cmd.Dir); err != nil {
		cmd.Dir = root
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "\nmemory scenario eval failed (target >= %.0f%%)\n", targetScore)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "\nmemory scenario eval: passed (target >= %.0f%%)\n", targetScore)
}
