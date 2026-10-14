package builtin

import (
	"testing"

	"github.com/coditary/taiji/internal/provider"
)

func TestFormatGenerateImageResult(t *testing.T) {
	text := formatGenerateImageResult("a red fox in snow", &provider.WujiImageResponse{
		Path:   "/tmp/wuji/out.png",
		Format: "png",
	})
	if !containsAll(text, "path: ", "/tmp/wuji/out.png", "markdown: ![a red fox in snow]") {
		t.Fatalf("unexpected output:\n%s", text)
	}
}

func TestFormatGenerateImageResultMultiplePaths(t *testing.T) {
	text := formatGenerateImageResult("tiles", &provider.WujiImageResponse{
		Paths:  []string{"/tmp/a.png", "/tmp/b.png"},
		Format: "png",
	})
	if !containsAll(text, "path: /tmp/a.png", "path[1]: /tmp/b.png") {
		t.Fatalf("unexpected output:\n%s", text)
	}
}

func TestIntArg(t *testing.T) {
	args := map[string]any{"width": float64(768), "height": 512}
	if intArg(args, "width") != 768 {
		t.Fatalf("width = %d", intArg(args, "width"))
	}
	if intArg(args, "height") != 512 {
		t.Fatalf("height = %d", intArg(args, "height"))
	}
	if intArg(args, "steps") != 0 {
		t.Fatalf("steps = %d", intArg(args, "steps"))
	}
}

func containsAll(s string, parts ...string) bool {
	for _, part := range parts {
		if !contains(s, part) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && (s == sub || indexOf(s, sub) >= 0))
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
