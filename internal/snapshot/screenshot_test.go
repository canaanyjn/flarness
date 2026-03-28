package snapshot

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewScreenshotter(t *testing.T) {
	s := NewScreenshotter("/test/project", "chrome", "ws://localhost:1234/ws", "/tmp/screenshots")
	if s.project != "/test/project" {
		t.Errorf("expected project /test/project, got %s", s.project)
	}
	if s.device != "chrome" {
		t.Errorf("expected device chrome, got %s", s.device)
	}
	if s.debugURL != "ws://localhost:1234/ws" {
		t.Errorf("expected debugURL ws://localhost:1234/ws, got %s", s.debugURL)
	}
}

func TestIsWebDevice(t *testing.T) {
	tests := []struct {
		device string
		want   bool
	}{
		{"chrome", true},
		{"Chrome", true},
		{"web-server", true},
		{"edge", true},
		{"web-renderer", true},
		{"iPhone", false},
		{"android", false},
		{"macos", false},
		{"linux", false},
		{"windows", false},
	}

	for _, tt := range tests {
		s := NewScreenshotter("", tt.device, "", "")
		got := s.isWebDevice()
		if got != tt.want {
			t.Errorf("isWebDevice(%q) = %v, want %v", tt.device, got, tt.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{2621440, "2.5MB"},
	}

	for _, tt := range tests {
		got := formatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestScreenshotDirCreation(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "flarness_test_screenshots")
	defer os.RemoveAll(tmpDir)

	s := NewScreenshotter("/test", "macos", "", tmpDir)

	// The directory shouldn't exist yet.
	if _, err := os.Stat(tmpDir); !os.IsNotExist(err) {
		// Clean up if it already exists.
		os.RemoveAll(tmpDir)
	}

	// Capture will fail (no flutter running), but dir should be created.
	_, _ = s.Capture()

	// The output dir should have been created.
	if _, err := os.Stat(tmpDir); os.IsNotExist(err) {
		t.Error("expected screenshot directory to be created")
	}
}

func TestResolveCDPURL(t *testing.T) {
	tests := []struct {
		debugURL string
	}{
		{"ws://localhost:1234/ws"},
		{"wss://localhost:1234/ws"},
		{"http://localhost:1234"},
	}

	for _, tt := range tests {
		s := NewScreenshotter("", "chrome", tt.debugURL, "")
		got := s.resolveCDPURL()
		if got == "" {
			t.Errorf("resolveCDPURL(%q) returned empty string", tt.debugURL)
		}
	}
}

func TestVMServiceURL(t *testing.T) {
	tests := []struct {
		name     string
		debugURL string
		want     string
	}{
		{
			name:     "ws url",
			debugURL: "ws://127.0.0.1:1234/abc=/ws",
			want:     "http://127.0.0.1:1234/abc=",
		},
		{
			name:     "wss url",
			debugURL: "wss://example.com/def/ws",
			want:     "https://example.com/def",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewScreenshotter("", "macos", tt.debugURL, "")
			if got := s.vmServiceURL(); got != tt.want {
				t.Fatalf("vmServiceURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
