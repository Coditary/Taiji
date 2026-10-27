package builtin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadFileBytesSmallAndLarge(t *testing.T) {
	dir := t.TempDir()
	small := filepath.Join(dir, "small.txt")
	if err := os.WriteFile(small, []byte("tiny"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := readFileBytes(small)
	if err != nil || string(got) != "tiny" {
		t.Fatalf("small: err=%v got=%q", err, got)
	}

	large := filepath.Join(dir, "large.bin")
	payload := strings.Repeat("x", mmapReadThreshold+1024)
	if err := os.WriteFile(large, []byte(payload), 0o644); err != nil {
		t.Fatal(err)
	}
	gotLarge, err := readFileBytes(large)
	if err != nil || len(gotLarge) != len(payload) {
		t.Fatalf("large: err=%v len=%d want %d", err, len(gotLarge), len(payload))
	}
	if string(gotLarge) != payload {
		t.Fatal("large content mismatch")
	}
}

func TestReadFileBytesMissing(t *testing.T) {
	if _, err := readFileBytes(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("expected error")
	}
}
