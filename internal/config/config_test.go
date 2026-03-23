package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()

	if cfg.Log.BufferSize != 1000 {
		t.Errorf("buffer_size: got %d, want 1000", cfg.Log.BufferSize)
	}
	if cfg.Log.RetentionDays != 7 {
		t.Errorf("retention_days: got %d, want 7", cfg.Log.RetentionDays)
	}
	if cfg.Log.MaxFileSize != "50MB" {
		t.Errorf("max_file_size: got %q, want 50MB", cfg.Log.MaxFileSize)
	}
	if cfg.Defaults.Device != "chrome" {
		t.Errorf("device: got %q, want chrome", cfg.Defaults.Device)
	}
	if !cfg.CDP.Enabled {
		t.Error("cdp.enabled should be true by default")
	}
	if cfg.CDP.Timeout != "10s" {
		t.Errorf("cdp.timeout: got %q, want 10s", cfg.CDP.Timeout)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use a temp directory as HOME to avoid polluting real config.
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	cfg := Default()
	cfg.Defaults.Device = "macOS"
	cfg.Log.BufferSize = 2000

	if err := Save(cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	// Verify file exists.
	path := filepath.Join(tmpDir, ".flarness", "config.yaml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("config.yaml should exist after save")
	}

	// Load it back.
	loaded := Load()
	if loaded.Defaults.Device != "macOS" {
		t.Errorf("device: got %q, want macOS", loaded.Defaults.Device)
	}
	if loaded.Log.BufferSize != 2000 {
		t.Errorf("buffer_size: got %d, want 2000", loaded.Log.BufferSize)
	}
}

func TestLoadNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpDir)
	defer os.Setenv("HOME", origHome)

	// Should return defaults when file doesn't exist.
	cfg := Load()
	if cfg.Defaults.Device != "chrome" {
		t.Errorf("should return default device, got %q", cfg.Defaults.Device)
	}
}
