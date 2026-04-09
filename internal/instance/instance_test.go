package instance

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionForProjectStable(t *testing.T) {
	project := "/tmp/project-a"
	if got, want := SessionForProject(project), SessionForProject(project); got != want {
		t.Fatalf("session mismatch: got %q want %q", got, want)
	}
	if got := SessionForProject(project); len(got) != 8 {
		t.Fatalf("session length = %d, want 8", len(got))
	}
}

func TestPathsForSession(t *testing.T) {
	paths := PathsForSession("abc12345")
	if filepath.Base(paths.InstanceDir) != "abc12345" {
		t.Fatalf("instance dir = %q", paths.InstanceDir)
	}
	if filepath.Base(paths.SocketPath) != "daemon.sock" {
		t.Fatalf("socket path = %q", paths.SocketPath)
	}
	if filepath.Base(paths.MetaPath) != "meta.json" {
		t.Fatalf("meta path = %q", paths.MetaPath)
	}
}

func TestSaveAndListMetas(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	meta := Meta{
		Session:     "abc12345",
		ProjectPath: "/tmp/project-a",
		ProjectName: "project-a",
		Device:      "chrome",
		CreatedAt:   "2026-01-01T00:00:00Z",
	}
	if err := SaveMeta(meta); err != nil {
		t.Fatalf("SaveMeta error: %v", err)
	}

	metas, err := ListMetas()
	if err != nil {
		t.Fatalf("ListMetas error: %v", err)
	}
	if len(metas) != 1 {
		t.Fatalf("len(metas) = %d, want 1", len(metas))
	}
	if metas[0].Session != meta.Session {
		t.Fatalf("session = %q, want %q", metas[0].Session, meta.Session)
	}
}

func TestCleanupAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	paths := PathsForSession("abc12345")
	if err := os.MkdirAll(paths.LogsDir, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := os.WriteFile(paths.MetaPath, []byte("{}"), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	if err := CleanupAll("abc12345"); err != nil {
		t.Fatalf("CleanupAll error: %v", err)
	}
	if _, err := os.Stat(paths.InstanceDir); !os.IsNotExist(err) {
		t.Fatalf("instance dir still exists: %v", err)
	}
}
