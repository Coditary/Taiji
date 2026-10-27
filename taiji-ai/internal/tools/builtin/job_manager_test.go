package builtin

import (
	"strings"
	"testing"
	"time"
)

func TestJobManagerStartLogsKill(t *testing.T) {
	mgr := NewJobManager()
	id, pid, err := mgr.Start("sess-1", "sh -c 'i=0; while [ $i -lt 50 ]; do echo line-$i; i=$((i+1)); sleep 0.05; done'", "")
	if err != nil {
		t.Fatal(err)
	}
	if pid <= 0 || id == "" {
		t.Fatalf("id=%s pid=%d", id, pid)
	}
	time.Sleep(200 * time.Millisecond)
	logs, err := mgr.Logs("sess-1", id, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs, "running") {
		t.Fatalf("expected running: %q", logs)
	}
	killMsg, err := mgr.Kill("sess-1", id)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(killMsg, "killed") {
		t.Fatalf("kill msg: %q", killMsg)
	}
	time.Sleep(100 * time.Millisecond)
	logs2, err := mgr.Logs("sess-1", id, 3)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(logs2, "exited") {
		t.Fatalf("expected exited: %q", logs2)
	}
}

func TestJobManagerStartRequiresCommand(t *testing.T) {
	mgr := NewJobManager()
	if _, _, err := mgr.Start("s", "  ", ""); err == nil {
		t.Fatal("expected empty command error")
	}
}

func TestJobManagerKillSession(t *testing.T) {
	mgr := NewJobManager()
	id, _, err := mgr.Start("s2", "sleep 60", "")
	if err != nil {
		t.Fatal(err)
	}
	mgr.KillSession("s2")
	logs, err := mgr.Logs("s2", id, 1)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(logs, "running") {
		t.Fatalf("job should be stopped: %q", logs)
	}
}
