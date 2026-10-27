package builtin

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "taiji-test-xdg-data-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_DATA_HOME", dir)
	configDir, err := os.MkdirTemp("", "taiji-test-xdg-config-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("XDG_CONFIG_HOME", configDir)
	os.Exit(m.Run())
}
