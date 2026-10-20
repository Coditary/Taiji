package stt

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
)

// Recorder captures microphone audio to a WAV file via ffmpeg.
type Recorder struct {
	mu   sync.Mutex
	cmd  *exec.Cmd
	path string
}

// Recording reports whether capture is active.
func (r *Recorder) Recording() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.cmd != nil
}

// LastPath returns the path of the last completed recording.
func (r *Recorder) LastPath() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.path
}

// Start begins recording into a new temp WAV file.
func (r *Recorder) Start(ctx context.Context, cfg taijicfg.STTConfig) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd != nil {
		return "", fmt.Errorf("already recording")
	}
	if err := requireRecorder(); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "taiji-stt-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "capture.wav")
	args, err := ffmpegRecordArgs(cfg, path)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("start recording: %w", err)
	}
	r.cmd = cmd
	r.path = path
	return path, nil
}

// Stop ends capture and returns the WAV path.
func (r *Recorder) Stop() (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.cmd == nil {
		if r.path != "" {
			return r.path, nil
		}
		return "", fmt.Errorf("not recording")
	}
	cmd := r.cmd
	r.cmd = nil
	if err := stopFFmpeg(cmd); err != nil {
		return "", err
	}
	if r.path == "" {
		return "", fmt.Errorf("recording path missing")
	}
	if _, err := os.Stat(r.path); err != nil {
		return "", fmt.Errorf("recording file: %w", err)
	}
	return r.path, nil
}

func stopFFmpeg(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return nil
	}
	_ = cmd.Process.Signal(os.Interrupt)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		if err != nil && !strings.Contains(err.Error(), "signal") {
			return err
		}
		return nil
	case <-time.After(4 * time.Second):
		_ = cmd.Process.Kill()
		return cmd.Wait()
	}
}

func requireRecorder() error {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		return fmt.Errorf("ffmpeg not found in PATH (required for microphone capture)")
	}
	return nil
}

// RecordFixedDuration captures a single microphone chunk (seconds).
func RecordFixedDuration(ctx context.Context, cfg taijicfg.STTConfig, duration time.Duration) (string, error) {
	if duration <= 0 {
		duration = 2 * time.Second
	}
	if err := requireRecorder(); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "taiji-stt-chunk-")
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "chunk.wav")
	args, err := ffmpegRecordArgs(cfg, path)
	if err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	sec := fmt.Sprintf("%.2f", duration.Seconds())
	// Insert -t before output path.
	args = append(args[:len(args)-1], "-t", sec, path)
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		_ = os.RemoveAll(dir)
		return "", fmt.Errorf("record chunk: %w", err)
	}
	return path, nil
}

func ffmpegRecordArgs(cfg taijicfg.STTConfig, path string) ([]string, error) {
	rate := cfg.SampleRate
	if rate <= 0 {
		rate = 16000
	}
	device := strings.TrimSpace(cfg.CaptureDevice)
	if device == "" {
		device = "default"
	}
	switch runtime.GOOS {
	case "darwin":
		input := device
		if input == "default" {
			input = ":0"
		}
		return []string{
			"ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
			"-f", "avfoundation", "-i", input,
			"-ac", "1", "-ar", fmt.Sprintf("%d", rate), path,
		}, nil
	case "windows":
		return []string{
			"ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
			"-f", "dshow", "-i", fmt.Sprintf("audio=%s", device),
			"-ac", "1", "-ar", fmt.Sprintf("%d", rate), path,
		}, nil
	default:
		return []string{
			"ffmpeg", "-hide_banner", "-loglevel", "error", "-y",
			"-f", "pulse", "-i", device,
			"-ac", "1", "-ar", fmt.Sprintf("%d", rate), path,
		}, nil
	}
}
