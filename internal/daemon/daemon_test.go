package daemon

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/canaanyjn/flarness/internal/instance"
)

func TestSessionForProject(t *testing.T) {
	// Same input should produce the same hash.
	h1 := instance.SessionForProject("/Users/canaanyjn/WorkSpace/Programming/Tools/FlutterHarness")
	h2 := instance.SessionForProject("/Users/canaanyjn/WorkSpace/Programming/Tools/FlutterHarness")
	if h1 != h2 {
		t.Errorf("same input produced different hashes: %s vs %s", h1, h2)
	}

	// Different input should produce different hashes.
	h3 := instance.SessionForProject("/Users/canaanyjn/other-project")
	if h1 == h3 {
		t.Errorf("different inputs produced the same hash: %s", h1)
	}

	// Hash should be 8 hex characters.
	if len(h1) != 8 {
		t.Errorf("hash length: got %d, want 8", len(h1))
	}
}

func TestPIDFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	d := &Daemon{
		session:    "abc12345",
		baseDir:    tmpDir,
		pidPath:    filepath.Join(tmpDir, "daemon.pid"),
		socketPath: filepath.Join(tmpDir, "daemon.sock"),
	}

	// Write PID file.
	if err := d.WritePID(); err != nil {
		t.Fatalf("WritePID error: %v", err)
	}

	// Read PID file.
	pid, err := d.ReadPID()
	if err != nil {
		t.Fatalf("ReadPID error: %v", err)
	}

	if pid != os.Getpid() {
		t.Errorf("PID: got %d, want %d", pid, os.Getpid())
	}

	// Cleanup should remove the PID file.
	d.Cleanup()
	if _, err := os.Stat(d.pidPath); !os.IsNotExist(err) {
		t.Error("PID file should be removed after Cleanup")
	}
}

func TestReadPIDNoFile(t *testing.T) {
	d := &Daemon{
		pidPath: filepath.Join(t.TempDir(), "nonexistent.pid"),
	}

	_, err := d.ReadPID()
	if err == nil {
		t.Error("expected error when reading nonexistent PID file")
	}
}

func TestIsRunningNoPID(t *testing.T) {
	d := &Daemon{
		pidPath: filepath.Join(t.TempDir(), "nonexistent.pid"),
	}

	if d.IsRunning() {
		t.Error("expected IsRunning=false when no PID file")
	}
}

func TestLogDir(t *testing.T) {
	tmpDir := t.TempDir()
	d := &Daemon{
		session: "abc12345",
		baseDir: tmpDir,
		project: "/Users/canaanyjn/my-flutter-project",
	}

	logDir := d.LogDir()
	if logDir == "" {
		t.Fatal("LogDir should not be empty")
	}
	if !filepath.IsAbs(logDir) {
		t.Errorf("LogDir should be absolute: %s", logDir)
	}
	expected := instance.PathsForSession("abc12345").LogsDir
	if logDir != expected {
		t.Errorf("LogDir should be %s, got %s", expected, logDir)
	}
}

func TestWriteProjectMeta(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	d := &Daemon{
		session: "abc12345",
		baseDir: instance.PathsForSession("abc12345").InstanceDir,
		project: "/Users/canaanyjn/my-flutter-project",
		device:  "chrome",
	}

	if err := d.WriteProjectMeta(); err != nil {
		t.Fatalf("WriteProjectMeta error: %v", err)
	}

	metaPath := instance.PathsForSession("abc12345").MetaPath
	data, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("cannot read meta.json: %v", err)
	}

	content := string(data)
	if !stringContains(content, "my-flutter-project") {
		t.Errorf("meta.json should contain project name, got: %s", content)
	}
	if !stringContains(content, "chrome") {
		t.Errorf("meta.json should contain device, got: %s", content)
	}
}

func TestStatus(t *testing.T) {
	d := &Daemon{
		session: "abc12345",
		project: "/test/project",
		device:  "chrome",
	}
	d.startTime = d.startTime // zero time

	status := d.Status()
	if status["project"] != "/test/project" {
		t.Errorf("project: got %v, want /test/project", status["project"])
	}
	if status["device"] != "chrome" {
		t.Errorf("device: got %v, want chrome", status["device"])
	}
	if status["session"] != "abc12345" {
		t.Errorf("session: got %v, want abc12345", status["session"])
	}
	if status["running"] != true {
		t.Errorf("running: got %v, want true", status["running"])
	}
}

func TestProcessExists(t *testing.T) {
	cmd := exec.Command("sleep", "2")
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start helper process: %v", err)
	}
	defer cmd.Process.Kill()

	if !processExists(cmd.Process) {
		t.Fatal("expected helper process to exist")
	}

	if err := cmd.Process.Signal(syscall.SIGKILL); err != nil {
		t.Fatalf("failed to kill helper process: %v", err)
	}
	_, _ = cmd.Process.Wait()

	if processExists(cmd.Process) {
		t.Fatal("expected helper process to be gone")
	}
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
