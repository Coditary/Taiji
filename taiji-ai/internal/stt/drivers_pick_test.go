package stt

import (
	"testing"

	wujicfg "github.com/coditary/wuji-core/pkg/config"
)

func TestPickPreferredDriver(t *testing.T) {
	drivers := []DriverOption{
		{ID: "ffmpeg", Name: "FFmpeg"},
		{ID: "llama", Name: "Llama"},
		{ID: "dummy", Name: "Dummy"},
	}
	if got := pickPreferredDriver(drivers); got != "llama" {
		t.Fatalf("pickPreferredDriver = %q, want llama", got)
	}
}

func TestMergeConfiguredDriversAddsLlama(t *testing.T) {
	wcfg := &wujicfg.Config{
		Drivers: wujicfg.DriversConfig{
			Registry: map[string]wujicfg.DriverSection{
				"llama": {DriverBin: "/usr/bin/wuji-driver-llama"},
			},
		},
	}
	merged := mergeConfiguredDrivers(nil, wcfg)
	if len(merged) != 1 || merged[0].ID != "llama" {
		t.Fatalf("mergeConfiguredDrivers = %+v", merged)
	}
}
