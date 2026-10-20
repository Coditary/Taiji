package config

import "strings"

// STTConfig configures speech-to-text via Wuji (wuji text --audio).
type STTConfig struct {
	Enabled       *bool   `yaml:"enabled,omitempty"`
	Driver        string  `yaml:"driver,omitempty"`
	Model         string  `yaml:"model,omitempty"`
	Language      string  `yaml:"language,omitempty"`
	Quality       string  `yaml:"quality,omitempty"`
	TrimSilence   *bool   `yaml:"trim_silence,omitempty"`
	SilenceLevel  float32 `yaml:"silence_level,omitempty"`
	CaptureDevice string  `yaml:"capture_device,omitempty"`
	SampleRate    int     `yaml:"sample_rate,omitempty"`
}

// DefaultSTTConfig returns built-in STT defaults.
func DefaultSTTConfig() STTConfig {
	enabled := true
	return STTConfig{
		Enabled:       &enabled,
		Language:      "auto",
		CaptureDevice: "default",
		SampleRate:    16000,
	}
}

// ResolvedSTT merges file config with defaults.
func (c *Config) ResolvedSTT() STTConfig {
	out := DefaultSTTConfig()
	if c == nil {
		return out
	}
	mergeSTT(&out, c.STT)
	return out
}

func mergeSTT(dst *STTConfig, src STTConfig) {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if strings.TrimSpace(src.Driver) != "" {
		dst.Driver = strings.TrimSpace(src.Driver)
	}
	if strings.TrimSpace(src.Model) != "" {
		dst.Model = strings.TrimSpace(src.Model)
	}
	if strings.TrimSpace(src.Language) != "" {
		dst.Language = strings.TrimSpace(src.Language)
	}
	if strings.TrimSpace(src.Quality) != "" {
		dst.Quality = strings.TrimSpace(src.Quality)
	}
	if src.TrimSilence != nil {
		dst.TrimSilence = src.TrimSilence
	}
	if src.SilenceLevel > 0 {
		dst.SilenceLevel = src.SilenceLevel
	}
	if strings.TrimSpace(src.CaptureDevice) != "" {
		dst.CaptureDevice = strings.TrimSpace(src.CaptureDevice)
	}
	if src.SampleRate > 0 {
		dst.SampleRate = src.SampleRate
	}
}

func (s STTConfig) EnabledOrDefault() bool {
	if s.Enabled == nil {
		return true
	}
	return *s.Enabled
}

func (s STTConfig) TrimSilenceOrDefault() bool {
	if s.TrimSilence == nil {
		return false
	}
	return *s.TrimSilence
}
