package snapshot

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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

func TestIsPNGData(t *testing.T) {
	if !isPNGData([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0x00}) {
		t.Fatal("expected valid PNG signature")
	}
	if isPNGData([]byte("skiapict")) {
		t.Fatal("skia picture must not be treated as PNG")
	}
}

func TestDetectScreenshotFormat(t *testing.T) {
	if got := detectScreenshotFormat([]byte("skiapict")); got != "skia picture" {
		t.Fatalf("detectScreenshotFormat() = %q", got)
	}
	if got := detectScreenshotFormat(nil); got != "empty data" {
		t.Fatalf("detectScreenshotFormat(nil) = %q", got)
	}
}

func TestCaptureFlutterFallbackUsesPlainScreenshot(t *testing.T) {
	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("shell unavailable")
	}
}

func TestShouldUseNativeDesktopScreenshot(t *testing.T) {
	if !shouldUseNativeDesktopScreenshot("macos", []byte("Screenshot not supported for macOS.")) {
		t.Fatal("expected native desktop fallback for macOS unsupported message")
	}
	if shouldUseNativeDesktopScreenshot("chrome", []byte("Screenshot not supported for macOS.")) {
		t.Fatal("web device must not use native desktop fallback")
	}
	if shouldUseNativeDesktopScreenshot("macos", []byte("some other flutter error")) {
		t.Fatal("unexpected native desktop fallback for unrelated error")
	}
}

func TestCaptureMacOSViaExtensionSuccess(t *testing.T) {
	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{
				Result: map[string]any{
					"result": string(mustJSON(t, map[string]any{
						"status":       "ok",
						"format":       "png",
						"image_base64": base64.StdEncoding.EncodeToString(validPNGData()),
						"width":        320,
						"height":       180,
						"pixel_ratio":  2.0,
					})),
				},
			}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter("/test", "macos", serverURL, t.TempDir())
	result, err := s.Capture()
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if result.Width != 320 || result.Height != 180 {
		t.Fatalf("unexpected dimensions: %+v", result)
	}
	if _, err := os.Stat(result.Path); err != nil {
		t.Fatalf("expected screenshot file: %v", err)
	}
}

func TestCaptureMacOSViaExtensionMissingPlugin(t *testing.T) {
	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{Error: "Unknown method \"ext.flarness.captureScreenshot\""}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter("/test", "macos", serverURL, t.TempDir())
	_, err := s.Capture()
	if err == nil {
		t.Fatal("expected missing extension error")
	}
	if !strings.Contains(err.Error(), "FlarnessPluginBinding.ensureInitialized()") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCaptureMacOSViaExtensionStatusError(t *testing.T) {
	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{
				Result: map[string]any{
					"result": string(mustJSON(t, map[string]any{
						"status": "error",
						"error":  "capture failed",
					})),
				},
			}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter("/test", "macos", serverURL, t.TempDir())
	_, err := s.Capture()
	if err == nil || !strings.Contains(err.Error(), "capture failed") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCaptureMacOSViaExtensionInvalidBase64(t *testing.T) {
	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{
				Result: map[string]any{
					"result": string(mustJSON(t, map[string]any{
						"status":       "ok",
						"format":       "png",
						"image_base64": "%%%not-base64%%%",
					})),
				},
			}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter("/test", "macos", serverURL, t.TempDir())
	_, err := s.Capture()
	if err == nil || !strings.Contains(err.Error(), "failed to decode macOS screenshot") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCaptureMacOSViaExtensionRejectsNonPNG(t *testing.T) {
	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{
				Result: map[string]any{
					"result": string(mustJSON(t, map[string]any{
						"status":       "ok",
						"format":       "png",
						"image_base64": base64.StdEncoding.EncodeToString([]byte("hello")),
					})),
				},
			}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter("/test", "macos", serverURL, t.TempDir())
	_, err := s.Capture()
	if err == nil || !strings.Contains(err.Error(), "non-PNG data") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCaptureFlutterFallsBackToExtensionOnCommandError(t *testing.T) {
	fakeFlutter := writeFakeFlutter(t, `#!/bin/sh
echo "flutter screenshot failed" >&2
exit 1
`)
	t.Setenv("PATH", filepath.Dir(fakeFlutter)+string(os.PathListSeparator)+os.Getenv("PATH"))

	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{
				Result: map[string]any{
					"result": string(mustJSON(t, map[string]any{
						"status":       "ok",
						"format":       "png",
						"image_base64": base64.StdEncoding.EncodeToString(validPNGData()),
						"width":        144,
						"height":       90,
						"pixel_ratio":  2.0,
					})),
				},
			}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter(t.TempDir(), "android", serverURL, t.TempDir())
	result, err := s.Capture()
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if result.Width != 144 || result.Height != 90 {
		t.Fatalf("unexpected dimensions: %+v", result)
	}
}

func TestCaptureFlutterFallsBackToExtensionOnInvalidPNG(t *testing.T) {
	fakeFlutter := writeFakeFlutter(t, `#!/bin/sh
out=""
prev=""
for arg in "$@"; do
  if [ "$prev" = "--out" ]; then
    out="$arg"
    break
  fi
  prev="$arg"
done
printf 'skiapict-invalid' > "$out"
exit 0
`)
	t.Setenv("PATH", filepath.Dir(fakeFlutter)+string(os.PathListSeparator)+os.Getenv("PATH"))

	serverURL := startFakeVMServiceServer(t, func(method string, _ map[string]any) rpcResponse {
		switch method {
		case "getVM":
			return rpcResponse{
				Result: map[string]any{
					"isolates": []map[string]any{
						{"id": "isolates/1", "name": "main"},
					},
				},
			}
		case "ext.flarness.captureScreenshot":
			return rpcResponse{
				Result: map[string]any{
					"result": string(mustJSON(t, map[string]any{
						"status":       "ok",
						"format":       "png",
						"image_base64": base64.StdEncoding.EncodeToString(validPNGData()),
						"width":        200,
						"height":       100,
						"pixel_ratio":  2.0,
					})),
				},
			}
		default:
			return rpcResponse{Error: "unexpected method: " + method}
		}
	})

	s := NewScreenshotter(t.TempDir(), "linux", serverURL, t.TempDir())
	result, err := s.Capture()
	if err != nil {
		t.Fatalf("Capture() error = %v", err)
	}
	if result.Width != 200 || result.Height != 100 {
		t.Fatalf("unexpected dimensions: %+v", result)
	}
}

type rpcResponse struct {
	Result any
	Error  string
}

func startFakeVMServiceServer(t *testing.T, handler func(method string, params map[string]any) rpcResponse) string {
	t.Helper()

	upgrader := websocket.Upgrader{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
		defer conn.Close()
		_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		for {
			var req struct {
				ID     any            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			if err := conn.ReadJSON(&req); err != nil {
				return
			}
			resp := handler(req.Method, req.Params)
			envelope := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
			}
			if resp.Error != "" {
				envelope["error"] = map[string]any{
					"code":    -32601,
					"message": resp.Error,
				}
			} else {
				envelope["result"] = resp.Result
			}
			if err := conn.WriteJSON(envelope); err != nil {
				t.Errorf("write failed: %v", err)
				return
			}
		}
	}))
	t.Cleanup(server.Close)

	return "ws" + strings.TrimPrefix(server.URL, "http")
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json marshal failed: %v", err)
	}
	return data
}

func validPNGData() []byte {
	return []byte{
		0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n',
		0x00, 0x00, 0x00, 0x0d, 'I', 'H', 'D', 'R',
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1f, 0x15, 0xc4,
		0x89, 0x00, 0x00, 0x00, 0x0a, 'I', 'D', 'A', 'T',
		0x78, 0x9c, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05,
		0x00, 0x01, 0x0d, 0x0a, 0x2d, 0xb4, 0x00, 0x00,
		0x00, 0x00, 'I', 'E', 'N', 'D', 0xae, 'B', 0x60, 0x82,
	}
}

func writeFakeFlutter(t *testing.T, script string) string {
	t.Helper()
	binDir := t.TempDir()
	path := filepath.Join(binDir, "flutter")
	if err := os.WriteFile(path, []byte(script), 0755); err != nil {
		t.Fatalf("write fake flutter: %v", err)
	}
	return path
}
