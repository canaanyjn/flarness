package collector

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

func TestNewCollector(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	if c.bufferSize != 100 {
		t.Errorf("bufferSize: got %d, want 100", c.bufferSize)
	}

	// current.jsonl should exist.
	if _, err := os.Stat(filepath.Join(dir, "current.jsonl")); os.IsNotExist(err) {
		t.Error("current.jsonl should be created")
	}
}

func TestDefaultBufferSize(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	if c.bufferSize != 1000 {
		t.Errorf("default bufferSize: got %d, want 1000", c.bufferSize)
	}
}

func TestAddAndQuery(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	// Add some entries.
	entries := []model.LogEntry{
		{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "hello"},
		{Timestamp: time.Now().UTC(), Level: model.LevelError, Source: model.SourceFramework, Message: "RenderBox error"},
		{Timestamp: time.Now().UTC(), Level: model.LevelWarning, Source: model.SourceTool, Message: "unused import"},
		{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "world"},
	}
	for _, e := range entries {
		c.Add(e)
	}

	// Query all.
	result := c.Query(QueryParams{Limit: 100})
	if len(result) != 4 {
		t.Fatalf("expected 4 results, got %d", len(result))
	}

	// Check stats.
	stats := c.Stats()
	if stats["total_logs"] != 4 {
		t.Errorf("total_logs: got %d, want 4", stats["total_logs"])
	}
	if stats["errors"] != 1 {
		t.Errorf("errors: got %d, want 1", stats["errors"])
	}
	if stats["warnings"] != 1 {
		t.Errorf("warnings: got %d, want 1", stats["warnings"])
	}
}

func TestQueryLevelFilter(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "info msg"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelError, Source: model.SourceApp, Message: "error msg"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelWarning, Source: model.SourceApp, Message: "warning msg"})

	result := c.Query(QueryParams{Level: "error"})
	if len(result) != 1 {
		t.Fatalf("expected 1 error, got %d", len(result))
	}
	if result[0].Message != "error msg" {
		t.Errorf("message: got %q", result[0].Message)
	}

	// Multiple levels.
	result = c.Query(QueryParams{Level: "error,warning"})
	if len(result) != 2 {
		t.Fatalf("expected 2 results for error,warning, got %d", len(result))
	}
}

func TestQuerySourceFilter(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "app msg"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceFramework, Message: "framework msg"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceTool, Message: "tool msg"})

	result := c.Query(QueryParams{Source: "app"})
	if len(result) != 1 {
		t.Fatalf("expected 1 app log, got %d", len(result))
	}

	result = c.Query(QueryParams{Source: "app,tool"})
	if len(result) != 2 {
		t.Fatalf("expected 2 results for app,tool, got %d", len(result))
	}
}

func TestQueryGrepFilter(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "hello world"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelError, Source: model.SourceFramework, Message: "RenderBox was not laid out"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "goodbye"})

	result := c.Query(QueryParams{Grep: "RenderBox"})
	if len(result) != 1 {
		t.Fatalf("expected 1 grep match, got %d", len(result))
	}

	// Regex pattern.
	result = c.Query(QueryParams{Grep: "Render.*not laid"})
	if len(result) != 1 {
		t.Fatalf("expected 1 regex match, got %d", len(result))
	}

	// No match.
	result = c.Query(QueryParams{Grep: "xyz_nonexistent"})
	if len(result) != 0 {
		t.Fatalf("expected 0 matches, got %d", len(result))
	}
}

func TestQuerySinceFilter(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	old := time.Now().Add(-10 * time.Minute).UTC()
	recent := time.Now().UTC()

	c.Add(model.LogEntry{Timestamp: old, Level: model.LevelInfo, Source: model.SourceApp, Message: "old msg"})
	c.Add(model.LogEntry{Timestamp: recent, Level: model.LevelInfo, Source: model.SourceApp, Message: "recent msg"})

	result := c.Query(QueryParams{Since: "5m"})
	if len(result) != 1 {
		t.Fatalf("expected 1 recent log, got %d", len(result))
	}
	if result[0].Message != "recent msg" {
		t.Errorf("message: got %q", result[0].Message)
	}
}

func TestQueryLimit(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	for i := 0; i < 20; i++ {
		c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "msg"})
	}

	result := c.Query(QueryParams{Limit: 5})
	if len(result) != 5 {
		t.Fatalf("expected 5 results with limit, got %d", len(result))
	}
}

func TestQueryCombinedFilters(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	now := time.Now().UTC()
	c.Add(model.LogEntry{Timestamp: now, Level: model.LevelError, Source: model.SourceFramework, Message: "RenderBox overflow"})
	c.Add(model.LogEntry{Timestamp: now, Level: model.LevelInfo, Source: model.SourceFramework, Message: "RenderBox info"})
	c.Add(model.LogEntry{Timestamp: now, Level: model.LevelError, Source: model.SourceApp, Message: "App overflow error"})

	result := c.Query(QueryParams{
		Level:  "error",
		Source: "framework",
		Grep:   "overflow",
	})

	if len(result) != 1 {
		t.Fatalf("expected 1 combined filter match, got %d", len(result))
	}
	if result[0].Message != "RenderBox overflow" {
		t.Errorf("message: got %q", result[0].Message)
	}
}

func TestBufferRotation(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 10})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	// Add 15 entries to a buffer of size 10.
	for i := 0; i < 15; i++ {
		c.Add(model.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     model.LevelInfo,
			Source:    model.SourceApp,
			Message:   "msg",
		})
	}

	// Buffer should not exceed bufferSize.
	stats := c.Stats()
	if stats["buffer"] > 10 {
		t.Errorf("buffer size: got %d, should not exceed 10", stats["buffer"])
	}
	// But total_logs should still count all.
	if stats["total_logs"] != 15 {
		t.Errorf("total_logs: got %d, want 15", stats["total_logs"])
	}
}

func TestJSONLPersistence(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}

	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "persisted msg"})
	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelError, Source: model.SourceApp, Message: "error msg"})
	c.Flush()
	c.Close()

	// Read the JSONL file.
	data, err := os.ReadFile(filepath.Join(dir, "current.jsonl"))
	if err != nil {
		t.Fatalf("read JSONL: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines in JSONL, got %d", len(lines))
	}

	// Verify each line is valid JSON.
	for i, line := range lines {
		var entry model.LogEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Errorf("line %d invalid JSON: %v", i, err)
		}
	}
}

func TestArchive(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	c.Add(model.LogEntry{Timestamp: time.Now().UTC(), Level: model.LevelInfo, Source: model.SourceApp, Message: "msg"})
	c.Flush()

	archivePath, err := c.Archive()
	if err != nil {
		t.Fatalf("Archive error: %v", err)
	}

	if archivePath == "" {
		t.Fatal("expected non-empty archive path")
	}

	// Archive file should exist.
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Error("archive file should exist")
	}

	// New current.jsonl should be fresh.
	if _, err := os.Stat(filepath.Join(dir, "current.jsonl")); os.IsNotExist(err) {
		t.Error("new current.jsonl should be created after archive")
	}
}

func TestListFiles(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	// Create some mock archive files.
	os.WriteFile(filepath.Join(dir, "2026-03-22_10-15-00.jsonl"), []byte("{}"), 0644)
	os.WriteFile(filepath.Join(dir, "2026-03-23_14-30-00.jsonl"), []byte("{}"), 0644)

	files, err := c.ListFiles()
	if err != nil {
		t.Fatalf("ListFiles error: %v", err)
	}

	// Should find at least current.jsonl + 2 archives.
	if len(files) < 3 {
		t.Errorf("expected at least 3 files, got %d", len(files))
	}
}

func TestCleanOld(t *testing.T) {
	dir := t.TempDir()
	c, err := New(Config{LogDir: dir, BufferSize: 100})
	if err != nil {
		t.Fatalf("New error: %v", err)
	}
	defer c.Close()

	// Create an "old" file by setting its modification time.
	oldPath := filepath.Join(dir, "2020-01-01_00-00-00.jsonl")
	os.WriteFile(oldPath, []byte("{}"), 0644)
	oldTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
	os.Chtimes(oldPath, oldTime, oldTime)

	// Create a "recent" file.
	recentPath := filepath.Join(dir, "2026-03-23_14-30-00.jsonl")
	os.WriteFile(recentPath, []byte("{}"), 0644)

	removed, err := c.CleanOld(7)
	if err != nil {
		t.Fatalf("CleanOld error: %v", err)
	}

	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}

	// Old file should be gone.
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("old file should be removed")
	}

	// Recent file should still exist.
	if _, err := os.Stat(recentPath); os.IsNotExist(err) {
		t.Error("recent file should still exist")
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		ok    bool
	}{
		{"30s", 30 * time.Second, true},
		{"5m", 5 * time.Minute, true},
		{"1h", 1 * time.Hour, true},
		{"2d", 48 * time.Hour, true},
		{"", 0, true},
		{"invalid", 0, false},
	}

	for _, tt := range tests {
		got, err := parseDuration(tt.input)
		if tt.ok && err != nil {
			t.Errorf("parseDuration(%q) error: %v", tt.input, err)
		}
		if !tt.ok && err == nil {
			t.Errorf("parseDuration(%q) expected error", tt.input)
		}
		if tt.ok && got != tt.want {
			t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
