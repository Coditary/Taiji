package stt

import (
	"strings"

	wujicfg "github.com/coditary/wuji-core/pkg/config"
	"github.com/coditary/wuji-core/pkg/driver/dummy"
)

// Drivers suited for wuji text --audio (speech-to-text).
var sttDriverPreference = []string{
	"llama",
	"whisper",
	"faster-whisper",
	"vllm",
	"openai",
	"groq",
	"mistral",
}

func isUndesiredSTTDriver(id string) bool {
	id = strings.TrimSpace(id)
	return id == "" || strings.EqualFold(id, dummy.DriverID)
}

func pickPreferredDriver(drivers []DriverOption) string {
	if len(drivers) == 0 {
		return ""
	}
	byID := map[string]DriverOption{}
	for _, d := range drivers {
		byID[d.ID] = d
	}
	for _, pref := range sttDriverPreference {
		if _, ok := byID[pref]; ok {
			return pref
		}
	}
	return drivers[0].ID
}

func mergeConfiguredDrivers(api []DriverOption, wcfg *wujicfg.Config) []DriverOption {
	out := append([]DriverOption(nil), api...)
	seen := map[string]struct{}{}
	for _, d := range out {
		seen[d.ID] = struct{}{}
	}
	if wcfg == nil || wcfg.Drivers.Registry == nil {
		return out
	}
	for id, section := range wcfg.Drivers.Registry {
		id = strings.TrimSpace(id)
		if isUndesiredSTTDriver(id) {
			continue
		}
		if strings.TrimSpace(section.DriverBin) == "" {
			continue
		}
		if !isSTTCandidateID(id) {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		name := id
		if strings.EqualFold(id, "llama") {
			name = "Llama (local STT)"
		}
		out = append(out, DriverOption{ID: id, Name: name})
	}
	return out
}

func isSTTCandidateID(id string) bool {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, pref := range sttDriverPreference {
		if id == pref {
			return true
		}
	}
	switch id {
	case "ffmpeg", "a1111", "local-rag", "raggo", "gorag", "laya", "dummy":
		return false
	default:
		return strings.Contains(id, "whisper") || strings.Contains(id, "llama") || strings.Contains(id, "vllm")
	}
}
