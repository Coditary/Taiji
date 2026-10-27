package session

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/coditary/taiji/internal/config"
)

func TestBuildSidebarIncludesEmptyFolder(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())

	root, err := filepath.Abs(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := config.RegisterProject(root, true); err != nil {
		t.Fatal(err)
	}

	sidebar, err := BuildSidebar(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sidebar.Folders) != 1 {
		t.Fatalf("folders = %d, want 1", len(sidebar.Folders))
	}
	if len(sidebar.Folders[0].Sessions) != 0 {
		t.Fatalf("sessions = %#v, want empty", sidebar.Folders[0].Sessions)
	}

	raw, err := MarshalSidebarJSON(sidebar)
	if err != nil {
		t.Fatal(err)
	}
	var decoded config.ProjectsSidebar
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Folders) != 1 || len(decoded.Folders[0].Sessions) != 0 {
		t.Fatalf("decoded folders = %#v", decoded.Folders)
	}
}
