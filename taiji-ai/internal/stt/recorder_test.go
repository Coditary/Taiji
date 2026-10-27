package stt

import (
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestRecorderLastPathWhenIdle(t *testing.T) {
	var r Recorder
	if r.Recording() {
		t.Fatal("expected not recording")
	}
	if r.LastPath() != "" {
		t.Fatalf("last path=%q", r.LastPath())
	}
}

func TestFFmpegRecordArgsLinuxDefault(t *testing.T) {
	args, err := ffmpegRecordArgs(taijicfg.STTConfig{}, "/tmp/out.wav")
	if err != nil {
		t.Fatal(err)
	}
	if len(args) < 2 || args[0] != "ffmpeg" {
		t.Fatalf("args=%v", args)
	}
	if args[len(args)-1] != "/tmp/out.wav" {
		t.Fatalf("missing output path: %v", args)
	}
}

func TestStopWithoutActiveRecording(t *testing.T) {
	var r Recorder
	if _, err := r.Stop(); err == nil {
		t.Fatal("expected error when not recording")
	}
}
