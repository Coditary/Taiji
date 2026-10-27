package provider

import (
	"bytes"
	"io"
	"os"
	"testing"

	taijicfg "github.com/coditary/taiji/internal/config"
)

func TestResolveTaijiWujiEndpointNilConfig(t *testing.T) {
	ep := resolveTaijiWujiEndpoint(nil)
	if ep == "" {
		t.Fatal("expected endpoint")
	}
	cfg := &taijicfg.Config{Root: t.TempDir(), Wuji: taijicfg.WujiConfig{Endpoint: "auto"}}
	if resolveTaijiWujiEndpoint(cfg) == "" {
		t.Fatal("expected endpoint from cfg")
	}
}

func TestLogSyncResultMessages(t *testing.T) {
	stderr := captureStderr(func() {
		LogSyncResult(SyncResult{})
		LogSyncResult(SyncResult{Message: "synced", Connected: true})
		LogSyncResult(SyncResult{Message: "offline", Connected: false})
	})
	if !bytes.Contains(stderr, []byte("synced")) || !bytes.Contains(stderr, []byte("offline")) {
		t.Fatalf("stderr=%q", stderr)
	}
}

func captureStderr(fn func()) []byte {
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	os.Stderr = w
	fn()
	_ = w.Close()
	os.Stderr = old
	out, _ := io.ReadAll(r)
	_ = r.Close()
	return out
}
