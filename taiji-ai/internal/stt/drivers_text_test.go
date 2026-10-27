package stt

import (
	"testing"

	"github.com/coditary/wuji-core/pkg/capability"
	wujidriver "github.com/coditary/wuji-core/pkg/driver"
)

func TestDriverSupportsText(t *testing.T) {
	if driverSupportsText(wujidriver.Info{}) {
		t.Fatal("expected false without capabilities")
	}
	if !driverSupportsText(wujidriver.Info{
		Capabilities: []capability.Type{capability.TextGeneration},
	}) {
		t.Fatal("expected true with text generation")
	}
}
