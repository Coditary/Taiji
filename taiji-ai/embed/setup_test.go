package main

import (
	"os"
	"testing"

	"github.com/coditary/taiji/internal/runtime"
)

func TestMain(m *testing.M) {
	data, err := os.MkdirTemp("", "taiji-test-xdg-data-*")
	if err != nil {
		panic(err)
	}
	config, err := os.MkdirTemp("", "taiji-test-xdg-config-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", data)
	_ = os.Setenv("XDG_CONFIG_HOME", config)
	os.Exit(m.Run())
}

// newTestRuntimeHandle marks the async runtime init as done (see getRuntime).
func newTestRuntimeHandle(rt *runtime.Runtime) *runtimeHandle {
	return &runtimeHandle{rt: rt, rtReady: true}
}
