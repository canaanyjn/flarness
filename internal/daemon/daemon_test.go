package daemon

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFnvHash(t *testing.T) {
	// Same input should produce the same hash.
	h1 := fnvHash("/Users/canaanyjn/WorkSpace/Programming/Tools/FlutterHarness")
	h2 := fnvHash("/Users/canaanyjn/WorkSpace/Programming/Tools/FlutterHarness")
	if h1 != h2 {
		t.Errorf("same input produced different hashes: %s vs %s", h1, h2)
	}

	// Different input should produce different hashes.
	h3 := fnvHash("/Users/canaanyjn/other-project")
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
	// Should be under baseDir/logs/.
	expected := filepath.Join(tmpDir, "logs")
	if !hasPrefix(logDir, expected) {
		t.Errorf("LogDir should be under %s, got %s", expected, logDir)
	}
}

func TestWriteProjectMeta(t *testing.T) {
	tmpDir := t.TempDir()
	d := &Daemon{
		baseDir: tmpDir,
		project: "/Users/canaanyjn/my-flutter-project",
		device:  "chrome",
	}

	if err := d.WriteProjectMeta(); err != nil {
		t.Fatalf("WriteProjectMeta error: %v", err)
	}

	metaPath := filepath.Join(d.LogDir(), "meta.json")
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
	if status["running"] != true {
		t.Errorf("running: got %v, want true", status["running"])
	}
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
