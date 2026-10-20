package stt

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	taijicfg "github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
	"github.com/coditary/wuji-core/pkg/capability"
	wujicfg "github.com/coditary/wuji-core/pkg/config"
	"github.com/coditary/wuji-core/pkg/core"
	wujidriver "github.com/coditary/wuji-core/pkg/driver"
	"github.com/coditary/wuji-core/pkg/driver/dummy"
)

// DriverOption is a selectable Wuji text driver for transcription.
type DriverOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ListDrivers returns Wuji drivers that advertise text generation.
func ListDrivers(ctx context.Context, cfg *taijicfg.Config) ([]DriverOption, error) {
	client, err := dialWuji(ctx, cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	infos, err := client.ListDrivers(ctx)
	if err != nil {
		return nil, err
	}
	var out []DriverOption
	seen := map[string]struct{}{}
	for _, info := range infos {
		if !driverSupportsText(info) {
			continue
		}
		id := strings.TrimSpace(info.ID)
		if id == "" || id == dummy.DriverID {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		name := strings.TrimSpace(info.Name)
		if name == "" {
			name = id
		}
		out = append(out, DriverOption{ID: id, Name: name})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	wcfg, werr := loadWujiConfig(cfg)
	if werr == nil {
		out = mergeConfiguredDrivers(out, wcfg)
		sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	}
	return out, nil
}

func driverSupportsText(info wujidriver.Info) bool {
	for _, cap := range info.Capabilities {
		if cap == capability.TextGeneration {
			return true
		}
	}
	return false
}

// ResolveDriver picks the configured driver or Wuji default for text capability.
func ResolveDriver(cfg *taijicfg.Config) string {
	if cfg == nil {
		return ""
	}
	stt := cfg.ResolvedSTT()
	if id := strings.TrimSpace(stt.Driver); id != "" && !isUndesiredSTTDriver(id) {
		return id
	}
	wujiCfg, err := loadWujiConfig(cfg)
	if err == nil && wujiCfg != nil {
		if cap := strings.TrimSpace(wujiCfg.DriverForCapability(capability.TextGeneration)); cap != "" && !isUndesiredSTTDriver(cap) {
			return cap
		}
		defaultDriver := strings.TrimSpace(wujiCfg.DefaultDriver)
		resolved := core.ResolveDriverID(wujiCfg, defaultDriver, "", capability.TextGeneration)
		if !isUndesiredSTTDriver(resolved) {
			return resolved
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	drivers, listErr := ListDrivers(ctx, cfg)
	if listErr == nil {
		if pick := pickPreferredDriver(drivers); pick != "" {
			return pick
		}
	}
	if id := strings.TrimSpace(stt.Driver); id != "" {
		return id
	}
	return ""
}

// TranscribeFile sends audio to Wuji GenerateText (same path as `wuji text --audio`).
func TranscribeFile(ctx context.Context, cfg *taijicfg.Config, audioPath string) (string, error) {
	if cfg == nil {
		return "", fmt.Errorf("config not loaded")
	}
	audioPath = strings.TrimSpace(audioPath)
	if audioPath == "" {
		return "", fmt.Errorf("empty audio path")
	}
	if _, err := os.Stat(audioPath); err != nil {
		return "", fmt.Errorf("audio file: %w", err)
	}
	sttCfg := cfg.ResolvedSTT()
	if !sttCfg.EnabledOrDefault() {
		return "", fmt.Errorf("speech input is disabled in settings")
	}
	driverID := ResolveDriver(cfg)
	if driverID == "" {
		return "", fmt.Errorf("no STT driver configured (set stt.driver in .taiji/config.yaml or Wuji capability_drivers.text)")
	}
	beam, err := ParseQuality(sttCfg.Quality)
	if err != nil {
		return "", err
	}
	trim := sttCfg.TrimSilenceOrDefault() || sttCfg.SilenceLevel > 0

	req := wujidriver.TextRequest{
		InputMode:    wujidriver.TextInputAudio,
		MediaPath:    audioPath,
		Language:     strings.TrimSpace(sttCfg.Language),
		Model:        strings.TrimSpace(sttCfg.Model),
		BeamSize:     beam,
		VADEnabled:   trim,
		VADThreshold: sttCfg.SilenceLevel,
	}

	client, err := dialWuji(ctx, cfg)
	if err != nil {
		return "", err
	}
	defer client.Close()
	if err := client.EnsureDriver(ctx, driverID); err != nil {
		return "", err
	}
	resp, err := client.GenerateText(ctx, driverID, req)
	if err != nil {
		return "", err
	}
	text := strings.TrimSpace(resp.Text)
	if text == "" {
		return "", fmt.Errorf("empty transcription")
	}
	return text, nil
}

func dialWuji(ctx context.Context, cfg *taijicfg.Config) (*provider.WujiCoreClient, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config not loaded")
	}
	root := provider.ResolveWujiRoot(cfg)
	ep := provider.ResolveTaijiWujiEndpoint(cfg.Root, cfg.Wuji.Endpoint, cfg.Wuji.WujiRoot)
	return provider.DialWujiCore(ctx, ep, root)
}

func loadWujiConfig(cfg *taijicfg.Config) (*wujicfg.Config, error) {
	root := provider.ResolveWujiRoot(cfg)
	prev, had := os.LookupEnv("WUJI_ROOT")
	os.Setenv("WUJI_ROOT", root)
	defer func() {
		if had {
			os.Setenv("WUJI_ROOT", prev)
		} else {
			os.Unsetenv("WUJI_ROOT")
		}
	}()
	wcfg, err := wujicfg.Load()
	if err != nil {
		return nil, err
	}
	return wcfg, nil
}
