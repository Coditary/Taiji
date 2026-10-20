package main

import (
	"context"
	"os"
	"strings"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/taiji/internal/stt"
	"github.com/coditary/wuji-core/pkg/capability"
	"github.com/coditary/wuji-core/pkg/driver/dummy"
	wujicfg "github.com/coditary/wuji-core/pkg/config"
)

const sttChunkInterval = 2 * time.Second

type sttToggleResult struct {
	Recording bool   `json:"recording"`
	Text      string `json:"text,omitempty"`
	Error     string `json:"error,omitempty"`
}

func (h *runtimeHandle) runSTTStream(ctx context.Context) {
	cfg := h.cfg.ResolvedSTT()
	accum := ""
	for {
		if ctx.Err() != nil {
			break
		}
		chunkCtx, cancel := context.WithTimeout(ctx, sttChunkInterval+3*time.Second)
		path, err := stt.RecordFixedDuration(chunkCtx, cfg, sttChunkInterval)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			continue
		}
		text, err := stt.TranscribeFile(ctx, h.cfg, path)
		_ = os.Remove(path)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			continue
		}
		accum = stt.AppendTranscript(accum, text)
		if accum != "" {
			h.emitSTTPartial(accum)
		}
	}
	h.sttMu.Lock()
	h.sttAccum = accum
	h.sttMu.Unlock()
}

func (h *runtimeHandle) sttToggle(ctx context.Context) sttToggleResult {
	if h == nil || h.cfg == nil {
		return sttToggleResult{Error: "runtime not ready"}
	}
	if !h.cfg.ResolvedSTT().EnabledOrDefault() {
		return sttToggleResult{Error: "speech input is disabled in settings"}
	}

	h.sttMu.Lock()
	if h.sttStreamCancel != nil {
		cancel := h.sttStreamCancel
		h.sttStreamCancel = nil
		h.sttMu.Unlock()
		cancel()
		if h.sttStreamDone != nil {
			select {
			case <-h.sttStreamDone:
			case <-time.After(30 * time.Second):
			}
		}
		h.sttMu.Lock()
		text := strings.TrimSpace(h.sttAccum)
		h.sttAccum = ""
		h.sttMu.Unlock()
		if text == "" {
			return sttToggleResult{Recording: false, Error: "no speech detected"}
		}
		return sttToggleResult{Recording: false, Text: text}
	}

	streamCtx, cancel := context.WithCancel(ctx)
	h.sttStreamCancel = cancel
	h.sttStreamDone = make(chan struct{})
	h.sttAccum = ""
	h.sttMu.Unlock()

	go func() {
		h.runSTTStream(streamCtx)
		close(h.sttStreamDone)
	}()

	return sttToggleResult{Recording: true}
}

func sttRecording(h *runtimeHandle) bool {
	if h == nil {
		return false
	}
	h.sttMu.Lock()
	defer h.sttMu.Unlock()
	return h.sttStreamCancel != nil
}

func fillSTTSettings(h *runtimeHandle, out *settingsPayload) {
	if h == nil || h.cfg == nil || out == nil {
		return
	}
	sttCfg := h.cfg.ResolvedSTT()
	out.STTEnabled = sttCfg.EnabledOrDefault()
	out.STTDriver = strings.TrimSpace(sttCfg.Driver)
	if out.STTDriver == "" || strings.EqualFold(out.STTDriver, "dummy") {
		out.STTDriver = stt.ResolveDriver(h.cfg)
	}
	out.STTModel = strings.TrimSpace(sttCfg.Model)
	out.STTLanguage = strings.TrimSpace(sttCfg.Language)
	if out.STTLanguage == "" {
		out.STTLanguage = "auto"
	}
	out.STTQuality = strings.TrimSpace(sttCfg.Quality)
	out.STTTrimSilence = sttCfg.TrimSilenceOrDefault()
	out.STTSilenceLevel = sttCfg.SilenceLevel
	out.STTCaptureDevice = strings.TrimSpace(sttCfg.CaptureDevice)
	if out.STTCaptureDevice == "" {
		out.STTCaptureDevice = "default"
	}
	out.STTSampleRate = sttCfg.SampleRate
	if out.STTSampleRate <= 0 {
		out.STTSampleRate = 16000
	}

	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	drivers, err := stt.ListDrivers(ctx, h.cfg)
	if err != nil {
		out.STTDriversError = err.Error()
	}
	for _, d := range drivers {
		out.STTDriverOptions = append(out.STTDriverOptions, d.ID)
		out.STTDriverLabels = append(out.STTDriverLabels, d.Name)
	}
	if len(out.STTDriverOptions) == 0 && out.STTDriversError == "" {
		out.STTDriversError = "no speech driver found — configure llama in ~/.wuji/config.yaml and set capability_drivers.text: llama"
	}
	if out.STTDriver == "" || strings.EqualFold(out.STTDriver, "dummy") {
		if pick := stt.ResolveDriver(h.cfg); pick != "" {
			out.STTDriver = pick
		}
	}
}

func applySTTSettings(cfg *taijicfg.Config, payload settingsPayload) {
	if cfg == nil {
		return
	}
	sttCfg := cfg.ResolvedSTT()
	enabled := payload.STTEnabled
	sttCfg.Enabled = &enabled
	sttCfg.Driver = strings.TrimSpace(payload.STTDriver)
	if isUndesiredSTTDriver(sttCfg.Driver) {
		sttCfg.Driver = stt.ResolveDriver(cfg)
	}
	sttCfg.Model = strings.TrimSpace(payload.STTModel)
	sttCfg.Language = strings.TrimSpace(payload.STTLanguage)
	sttCfg.Quality = strings.TrimSpace(payload.STTQuality)
	trim := payload.STTTrimSilence
	sttCfg.TrimSilence = &trim
	sttCfg.SilenceLevel = payload.STTSilenceLevel
	sttCfg.CaptureDevice = strings.TrimSpace(payload.STTCaptureDevice)
	if payload.STTSampleRate > 0 {
		sttCfg.SampleRate = payload.STTSampleRate
	}
	cfg.STT = sttCfg
	syncWujiTextDriverForSTT(cfg, sttCfg.Driver)
}

func isUndesiredSTTDriver(id string) bool {
	id = strings.TrimSpace(id)
	return id == "" || strings.EqualFold(id, dummy.DriverID)
}

func syncWujiTextDriverForSTT(cfg *taijicfg.Config, driverID string) {
	if cfg == nil || isUndesiredSTTDriver(driverID) {
		return
	}
	root := provider.ResolveWujiRoot(cfg)
	if root == "" {
		return
	}
	_ = wujicfg.SetCapabilityDriver(root, string(capability.TextGeneration), driverID)
}
