package snapshot

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Screenshotter captures screenshots from the running Flutter app.
type Screenshotter struct {
	mu         sync.Mutex
	project    string
	device     string
	debugURL   string // VM Service ws:// URL
	outputDir  string
}

// ScreenshotResult holds the result of a screenshot capture.
type ScreenshotResult struct {
	Path   string `json:"path"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
	Size   string `json:"size"`
	Device string `json:"device"`
}

// NewScreenshotter creates a new Screenshotter.
func NewScreenshotter(project, device, debugURL, outputDir string) *Screenshotter {
	return &Screenshotter{
		project:  project,
		device:   device,
		debugURL: debugURL,
		outputDir: outputDir,
	}
}

// Capture takes a screenshot and saves it to the output directory.
func (s *Screenshotter) Capture() (*ScreenshotResult, error) {
	// Ensure output directory exists.
	if err := os.MkdirAll(s.outputDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create screenshot dir: %w", err)
	}

	filename := fmt.Sprintf("screenshot_%s.png", time.Now().Format("2006-01-02_15-04-05"))
	outPath := filepath.Join(s.outputDir, filename)

	if s.isWebDevice() {
		return s.captureCDP(outPath)
	}
	return s.captureFlutter(outPath)
}

// captureCDP uses Chrome DevTools Protocol to capture a screenshot.
func (s *Screenshotter) captureCDP(outPath string) (*ScreenshotResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.debugURL == "" {
		return nil, fmt.Errorf("no debug URL available for CDP screenshot")
	}

	// Connect to the page target via CDP.
	// The debugURL from flutter is a VM service URL.
	// We need the browser's CDP endpoint.
	cdpURL := s.resolveCDPURL()

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(cdpURL, nil)
	if err != nil {
		// Fallback to flutter screenshot command.
		return s.captureFlutter(outPath)
	}
	defer conn.Close()

	// Send Page.captureScreenshot command.
	reqID := 1
	req := map[string]any{
		"id":     reqID,
		"method": "Page.captureScreenshot",
		"params": map[string]any{
			"format":  "png",
			"quality": 100,
		},
	}

	if err := conn.WriteJSON(req); err != nil {
		conn.Close()
		return s.captureFlutter(outPath)
	}

	// Read response with timeout.
	conn.SetReadDeadline(time.Now().Add(15 * time.Second))
	_, message, err := conn.ReadMessage()
	if err != nil {
		conn.Close()
		return s.captureFlutter(outPath)
	}

	var resp struct {
		ID     int `json:"id"`
		Result struct {
			Data string `json:"data"` // base64-encoded PNG
		} `json:"result"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(message, &resp); err != nil {
		return s.captureFlutter(outPath)
	}

	if resp.Error != nil {
		return s.captureFlutter(outPath)
	}

	if resp.Result.Data == "" {
		return s.captureFlutter(outPath)
	}

	// Decode base64 and write to file.
	imgData, err := base64.StdEncoding.DecodeString(resp.Result.Data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode screenshot: %w", err)
	}

	if err := os.WriteFile(outPath, imgData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write screenshot: %w", err)
	}

	return &ScreenshotResult{
		Path:   outPath,
		Size:   formatSize(int64(len(imgData))),
		Device: s.device,
	}, nil
}

// captureFlutter uses the `flutter screenshot` command for non-web platforms.
func (s *Screenshotter) captureFlutter(outPath string) (*ScreenshotResult, error) {
	args := []string{"screenshot", "--out", outPath}

	// If we have a debug URL, use the VM service observatory.
	if s.debugURL != "" {
		// Convert ws:// to http:// for observatory URL.
		obsURL := s.debugURL
		obsURL = strings.Replace(obsURL, "ws://", "http://", 1)
		obsURL = strings.Replace(obsURL, "wss://", "https://", 1)
		// Remove /ws suffix if present.
		obsURL = strings.TrimSuffix(obsURL, "/ws")
		args = append(args, "--observatory-url", obsURL)
	}

	cmd := exec.Command("flutter", args...)
	cmd.Dir = s.project

	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("flutter screenshot failed: %w\nOutput: %s", err, string(output))
	}

	// Check file was created.
	info, err := os.Stat(outPath)
	if err != nil {
		return nil, fmt.Errorf("screenshot file not found after capture")
	}

	return &ScreenshotResult{
		Path:   outPath,
		Size:   formatSize(info.Size()),
		Device: s.device,
	}, nil
}

// resolveCDPURL tries to get the CDP page endpoint from the debug URL.
func (s *Screenshotter) resolveCDPURL() string {
	// The flutter debug URL is typically a VM service URL.
	// For Web, we try the same host/port with /json to discover page targets.
	wsURL := s.debugURL
	if strings.HasPrefix(wsURL, "ws://") {
		wsURL = strings.Replace(wsURL, "ws://", "http://", 1)
	}

	// Try to get page target list.
	// First, just use the debug URL directly as it might be the page target already.
	return s.debugURL
}

// isWebDevice checks if the current device is a web browser.
func (s *Screenshotter) isWebDevice() bool {
	device := strings.ToLower(s.device)
	return device == "chrome" || device == "web-server" || device == "edge" || strings.Contains(device, "web")
}

// formatSize returns a human-readable file size.
func formatSize(bytes int64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}
