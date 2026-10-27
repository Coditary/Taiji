package grpcserver

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "taiji-test-xdg-data-*")
	if err != nil {
		panic(err)
	}
	config, err := os.MkdirTemp("", "taiji-test-xdg-config-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", dir)
	_ = os.Setenv("XDG_CONFIG_HOME", config)
	os.Exit(m.Run())
}
