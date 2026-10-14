package agent

import "testing"

func TestIsDecisionTool(t *testing.T) {
	if !isDecisionTool("decide") {
		t.Fatal("expected decide")
	}
	if isDecisionTool("list_dir") {
		t.Fatal("list_dir is not a decision tool")
	}
}
