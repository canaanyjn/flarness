package model

import (
	"encoding/json"
	"testing"
	"time"
)

func TestLogEntryJSON(t *testing.T) {
	entry := LogEntry{
		Timestamp:  time.Date(2026, 3, 23, 14, 30, 0, 0, time.UTC),
		Level:      LevelError,
		Source:     SourceFramework,
		Message:    "RenderBox was not laid out",
		StackTrace: "at Widget.build (widget.dart:42)",
		File:       "lib/pages/home_page.dart",
		Line:       42,
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded LogEntry
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Level != LevelError {
		t.Errorf("level: got %q, want %q", decoded.Level, LevelError)
	}
	if decoded.Source != SourceFramework {
		t.Errorf("source: got %q, want %q", decoded.Source, SourceFramework)
	}
	if decoded.Message != "RenderBox was not laid out" {
		t.Errorf("message mismatch: %q", decoded.Message)
	}
	if decoded.StackTrace != "at Widget.build (widget.dart:42)" {
		t.Errorf("stack mismatch: %q", decoded.StackTrace)
	}
	if decoded.File != "lib/pages/home_page.dart" {
		t.Errorf("file mismatch: %q", decoded.File)
	}
	if decoded.Line != 42 {
		t.Errorf("line: got %d, want 42", decoded.Line)
	}
}

func TestLogEntryOmitEmpty(t *testing.T) {
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     LevelInfo,
		Source:    SourceApp,
		Message:   "hello",
	}

	data, err := json.Marshal(entry)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	// stack, file should be omitted.
	if contains(raw, "stack") {
		t.Errorf("expected stack to be omitted, got: %s", raw)
	}
	if contains(raw, "file") {
		t.Errorf("expected file to be omitted, got: %s", raw)
	}
}

func TestLogConstants(t *testing.T) {
	levels := []string{LevelDebug, LevelInfo, LevelWarning, LevelError, LevelFatal}
	expected := []string{"debug", "info", "warning", "error", "fatal"}
	for i, l := range levels {
		if l != expected[i] {
			t.Errorf("level[%d]: got %q, want %q", i, l, expected[i])
		}
	}

	sources := []string{SourceApp, SourceFramework, SourceEngine, SourceTool}
	expectedSrc := []string{"app", "framework", "engine", "tool"}
	for i, s := range sources {
		if s != expectedSrc[i] {
			t.Errorf("source[%d]: got %q, want %q", i, s, expectedSrc[i])
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && jsonContains(s, sub)
}

func jsonContains(s, key string) bool {
	return len(s) > 0 && len(key) > 0 && stringContains(s, `"`+key+`"`)
}

func stringContains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
