package provider

import (
	"context"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestSyncProvidersFromWujiDisabled(t *testing.T) {
	cfg := &taijicfg.Config{Root: t.TempDir()}
	res, err := SyncProvidersFromWuji(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	if res.Connected || res.Message == "" {
		t.Fatalf("res=%+v", res)
	}
}

func TestRefreshProvidersFromWujiNilConfig(t *testing.T) {
	if _, err := RefreshProvidersFromWuji(context.Background(), nil); err == nil {
		t.Fatal("expected error")
	}
}
