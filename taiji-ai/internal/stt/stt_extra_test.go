package stt

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestTranscribeFileNoDriver(t *testing.T) {
	root := t.TempDir()
	badWujiRoot := filepath.Join(root, "no-wuji")
	t.Setenv("WUJI_ROOT", badWujiRoot)
	cfg := &taijicfg.Config{
		Root: root,
		STT:  taijicfg.STTConfig{Driver: "not-a-real-stt-driver"},
		Wuji: taijicfg.WujiConfig{Endpoint: "127.0.0.1:1", WujiRoot: badWujiRoot},
	}
	path := filepath.Join(t.TempDir(), "a.wav")
	if err := os.WriteFile(path, []byte("RIFF"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := TranscribeFile(context.Background(), cfg, path)
	if err == nil {
		t.Fatal("expected transcription error")
	}
	low := strings.ToLower(err.Error())
	if !strings.Contains(low, "driver") && !strings.Contains(low, "connect") && !strings.Contains(low, "wuji") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTranscribeFileInvalidQuality(t *testing.T) {
	cfg := &taijicfg.Config{
		Root: t.TempDir(),
		STT:  taijicfg.STTConfig{Driver: "whisper", Quality: "not-a-number"},
	}
	path := filepath.Join(t.TempDir(), "a.wav")
	if err := os.WriteFile(path, []byte("RIFF"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := TranscribeFile(context.Background(), cfg, path)
	if err == nil {
		t.Fatal("expected quality parse error")
	}
}

func TestLoadWujiConfigSetsEnv(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".wuji"), 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := &taijicfg.Config{Root: root, Wuji: taijicfg.WujiConfig{WujiRoot: root}}
	wcfg, err := loadWujiConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if wcfg == nil {
		t.Fatal("nil wuji config")
	}
}

func TestDialWujiNilConfig(t *testing.T) {
	if _, err := dialWuji(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}

func TestListDriversUnreachable(t *testing.T) {
	root := t.TempDir()
	badWujiRoot := filepath.Join(root, "no-wuji")
	t.Setenv("WUJI_ROOT", badWujiRoot)
	cfg := &taijicfg.Config{
		Root: root,
		Wuji: taijicfg.WujiConfig{Endpoint: "127.0.0.1:1", WujiRoot: badWujiRoot},
	}
	_, err := ListDrivers(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected dial error")
	}
}

func TestRequireRecorderMessage(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		if e := requireRecorder(); e == nil || !strings.Contains(e.Error(), "ffmpeg") {
			t.Fatalf("err=%v", e)
		}
	}
}
