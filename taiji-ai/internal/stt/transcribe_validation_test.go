package stt

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestTranscribeFileValidation(t *testing.T) {
	if _, err := TranscribeFile(context.Background(), nil, "x.wav"); err == nil {
		t.Fatal("nil config")
	}
	cfg := &taijicfg.Config{Root: t.TempDir()}
	if _, err := TranscribeFile(context.Background(), cfg, ""); err == nil {
		t.Fatal("empty path")
	}
	missing := filepath.Join(t.TempDir(), "missing.wav")
	if _, err := TranscribeFile(context.Background(), cfg, missing); err == nil || !strings.Contains(err.Error(), "audio file") {
		t.Fatalf("missing file: %v", err)
	}
}

func TestTranscribeFileDisabled(t *testing.T) {
	off := false
	cfg := &taijicfg.Config{
		Root: t.TempDir(),
		STT:  taijicfg.STTConfig{Enabled: &off},
	}
	path := filepath.Join(t.TempDir(), "a.wav")
	if err := os.WriteFile(path, []byte("RIFF"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := TranscribeFile(context.Background(), cfg, path); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled: %v", err)
	}
}

func TestResolveDriverExplicit(t *testing.T) {
	cfg := &taijicfg.Config{
		Root: t.TempDir(),
		STT:  taijicfg.STTConfig{Driver: "whisper-local"},
	}
	if got := ResolveDriver(cfg); got != "whisper-local" {
		t.Fatalf("got %q", got)
	}
	if ResolveDriver(nil) != "" {
		t.Fatal("nil config")
	}
}

func TestIsSTTCandidateAndUndesired(t *testing.T) {
	if !isUndesiredSTTDriver("dummy") || !isUndesiredSTTDriver("") {
		t.Fatal("undesired")
	}
	if !isSTTCandidateID("faster-whisper") || isSTTCandidateID("ffmpeg") {
		t.Fatal("candidate")
	}
}
