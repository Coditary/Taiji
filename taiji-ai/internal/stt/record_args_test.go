package stt

import (
	"runtime"
	"strings"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestFFmpegRecordArgsDefaults(t *testing.T) {
	args, err := ffmpegRecordArgs(taijicfg.STTConfig{}, "/tmp/out.wav")
	if err != nil {
		t.Fatal(err)
	}
	if args[0] != "ffmpeg" {
		t.Fatalf("args=%v", args)
	}
	joined := strings.Join(args, " ")
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(joined, "avfoundation") {
			t.Fatalf("darwin args=%v", args)
		}
	case "windows":
		if !strings.Contains(joined, "dshow") {
			t.Fatalf("windows args=%v", args)
		}
	default:
		if !strings.Contains(joined, "pulse") {
			t.Fatalf("linux args=%v", args)
		}
	}
	if !strings.Contains(joined, "16000") {
		t.Fatal("expected default sample rate")
	}
}

func TestRecorderNotRecordingInitially(t *testing.T) {
	var r Recorder
	if r.Recording() {
		t.Fatal("expected idle recorder")
	}
	if _, err := r.Stop(); err == nil {
		t.Fatal("stop without start should error")
	}
}
