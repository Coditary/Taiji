package provider

import (
	"errors"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestIsTransientWujiErr(t *testing.T) {
	if !isTransientWujiErr(errors.New("rpc error: code = Unavailable")) {
		t.Fatal("expected transient")
	}
	if isTransientWujiErr(errors.New("invalid model")) {
		t.Fatal("expected permanent")
	}
}

func TestResolveWujiEndpointExplicit(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir()}
	ep := resolveWujiEndpoint(taijicfg.ProviderConfig{Endpoint: "unix:///tmp/wuji.sock"}, cfg)
	if ep != "unix:///tmp/wuji.sock" {
		t.Fatalf("ep=%q", ep)
	}
}

func TestDialTargetUnixPrefix(t *testing.T) {
	if got := dialTarget("unix:///run/wuji.sock"); got != "unix:/run/wuji.sock" {
		t.Fatalf("got %q", got)
	}
}

func TestNewWujiProviderDefaults(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{}}
	p, err := newWujiProvider("wuji-main", taijicfg.ProviderConfig{Type: "wuji", Model: "test-model"}, cfg)
	if err != nil {
		t.Fatal(err)
	}
	wp := p.(*wujiProvider)
	if wp.driverID != "wuji-main" || wp.model != "test-model" {
		t.Fatalf("provider=%+v", wp)
	}
}
