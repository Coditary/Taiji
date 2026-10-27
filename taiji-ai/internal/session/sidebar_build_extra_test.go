package session

import (
	"testing"

	"github.com/coditary/taiji/internal/config"
	"github.com/coditary/taiji/internal/provider"
)

func TestBuildSidebarIncludesSessionRow(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	root := t.TempDir()
	if err := config.RegisterProject(root, true); err != nil {
		t.Fatal(err)
	}
	s, err := New(root, "row1", "p", "g")
	if err != nil {
		t.Fatal(err)
	}
	s.State.Messages = append(s.State.Messages, provider.Message{Role: provider.RoleUser, Content: "hello"})
	if err := s.Save(); err != nil {
		t.Fatal(err)
	}
	sidebar, err := BuildSidebar(root, s.State.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(sidebar.Folders) != 1 || len(sidebar.Folders[0].Sessions) != 1 {
		t.Fatalf("sidebar=%#v", sidebar.Folders)
	}
	if sidebar.CurrentSession != "row1" {
		t.Fatalf("current=%q", sidebar.CurrentSession)
	}
}
