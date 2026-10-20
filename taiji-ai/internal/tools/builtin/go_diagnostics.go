package builtin

import (
	"bytes"
	"context"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const goVetTimeout = 5 * time.Second

func enrichResultWithGoDiagnostics(path string, text string) string {
	if !strings.HasSuffix(strings.ToLower(path), ".go") {
		return text
	}
	note := runGoDiagnostics(path)
	if note == "" {
		return text
	}
	return text + "\n\n" + note
}

func runGoDiagnostics(path string) string {
	var parts []string
	if toolsSettings.AutoFormatOrDefault() {
		if out := runGofmt(path); out != "" {
			parts = append(parts, out)
		}
	}
	if msg := goSyntaxDiagnostics(path); msg != "" {
		parts = append(parts, msg)
	}
	if toolsSettings.AutoVetOrDefault() {
		if out := runGoVetPackage(path); out != "" {
			parts = append(parts, out)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return "[DIAGNOSTICS WARNING]\n" + strings.Join(parts, "\n")
}

func runGofmt(path string) string {
	cmd := exec.Command("gofmt", "-w", path)
	out, err := cmd.CombinedOutput()
	if err == nil {
		return ""
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		text = err.Error()
	}
	return "gofmt: " + text
}

func goSyntaxDiagnostics(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "parse: " + err.Error()
	}
	fset := token.NewFileSet()
	_, err = parser.ParseFile(fset, path, raw, parser.AllErrors)
	if err == nil {
		return ""
	}
	return "go/parser: " + err.Error()
}

func runGoVetPackage(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	dir := filepath.Dir(abs)
	if !hasGoSourceInDir(dir) {
		return ""
	}
	ctx, cancel := context.WithTimeout(context.Background(), goVetTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "vet", ".")
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err = cmd.Run()
	out := strings.TrimSpace(buf.String())
	if err == nil {
		return ""
	}
	if ctx.Err() != nil {
		return "go vet: timed out after " + goVetTimeout.String()
	}
	if out == "" {
		out = err.Error()
	}
	return "go vet: " + out
}

func hasGoSourceInDir(dir string) bool {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".go") {
			return true
		}
	}
	return false
}
