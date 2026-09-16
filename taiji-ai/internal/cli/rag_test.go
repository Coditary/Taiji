package cli

import (
	"reflect"
	"testing"
)

func TestDefaultIndexExcludes(t *testing.T) {
	got := defaultIndexExcludes([]string{"."}, nil)
	want := []string{"node_modules", ".git", ".wuji", ".taiji", ".cache", "vendor"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	custom := []string{"vendor"}
	if !reflect.DeepEqual(defaultIndexExcludes([]string{"."}, custom), custom) {
		t.Fatal("user excludes should win")
	}
	if defaultIndexExcludes([]string{"./docs"}, nil) != nil {
		t.Fatal("non-root paths should not get default excludes")
	}
}
