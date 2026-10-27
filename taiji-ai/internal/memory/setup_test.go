package memory

import (
	"os"
	"testing"
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
