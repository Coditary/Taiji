package cli

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	mustSetTempXDG()
	os.Exit(m.Run())
}

func mustSetTempXDG() {
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
}
