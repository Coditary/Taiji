package builtin

import (
	"path/filepath"
	"testing"
)

func TestIsSkippedPathAndSegments(t *testing.T) {
	if !isSkippedPath("node_modules") || isSkippedPath("src") {
		t.Fatal("isSkippedPath mismatch")
	}
	if !pathHasSkippedSegment(filepath.Join("proj", ".git", "objects")) {
		t.Fatal("expected skipped segment in path")
	}
	if pathHasSkippedSegment("src/main.go") {
		t.Fatal("did not expect skip in normal path")
	}
}

func TestMatchGlobDoubleStarSuffix(t *testing.T) {
	ok, err := matchGlob("nested/pkg/main.go", "**/*.go")
	if err != nil || !ok {
		t.Fatalf("match **/*.go: ok=%v err=%v", ok, err)
	}
}
