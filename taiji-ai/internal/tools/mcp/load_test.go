package mcp

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMCPServersMergesJSON(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "a.json")
	second := filepath.Join(dir, "b.json")
	if err := os.WriteFile(first, []byte(`{"mcpServers":{"one":{"command":"echo"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(`{"mcpServers":{"two":{"command":"cat"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	merged, err := loadMCPServersFromPaths([]string{first, second})
	if err != nil {
		t.Fatal(err)
	}
	if merged["one"].Command != "echo" || merged["two"].Command != "cat" {
		t.Fatalf("merged=%v", merged)
	}
}
