package daemon

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/canaanyjn/flarness/internal/analyzer"
	"github.com/canaanyjn/flarness/internal/collector"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/canaanyjn/flarness/internal/snapshot"
)

// Handler routes IPC commands to the appropriate logic.
type Handler struct {
	daemon *Daemon
}

// NewHandler creates a new command handler.
func NewHandler(d *Daemon) *Handler {
	return &Handler{daemon: d}
}

// Handle processes a command and returns a response.
func (h *Handler) Handle(cmd model.Command) model.Response {
	switch cmd.Cmd {
	case "status":
		return h.handleStatus()
	case "stop":
		return h.handleStop()
	case "reload":
		return h.handleReload()
	case "restart":
		return h.handleRestart()
	case "logs":
		return h.handleLogs(cmd.Args)
	case "analyze":
		return h.handleAnalyze()
	case "screenshot":
		return h.handleScreenshot()
	case "inspect":
		return h.handleInspect(cmd.Args)
	case "semantics":
		return h.handleSemantics(cmd.Args)
	case "tap":
		return h.handleTap(cmd.Args)
	case "type":
		return h.handleType(cmd.Args)
	case "scroll":
		return h.handleScroll(cmd.Args)
	case "swipe":
		return h.handleSwipe(cmd.Args)
	case "longpress":
		return h.handleLongPress(cmd.Args)
	case "wait":
		return h.handleWait(cmd.Args)
	default:
		return model.Response{
			OK:    false,
			Error: "unknown command: " + cmd.Cmd,
		}
	}
}

func (h *Handler) handleStatus() model.Response {
	status := h.daemon.Status()
	if h.daemon.procMgr != nil {
		status["flutter_state"] = h.daemon.procMgr.GetState().String()
		if h.daemon.procMgr.AppURL != "" {
			status["url"] = h.daemon.procMgr.AppURL
		}
	}
	return model.Response{
		OK:   true,
		Data: status,
	}
}

func (h *Handler) handleStop() model.Response {
	// Stop the Flutter process first.
	if h.daemon.procMgr != nil {
		h.daemon.procMgr.Stop()
	}

	// The actual daemon shutdown happens in server.go after sending this response.
	return model.Response{
		OK: true,
		Data: model.StopResponse{
			Status:  "ok",
			Message: "daemon stopping",
		},
	}
}

func (h *Handler) handleReload() model.Response {
	if h.daemon.procMgr == nil {
		return model.Response{
			OK:    false,
			Error: "flutter process not running",
		}
	}

	startTime := time.Now()

	if err := h.daemon.procMgr.SendReload(); err != nil {
		return model.Response{
			OK:    false,
			Error: err.Error(),
		}
	}

	// Wait for reload result with 60s timeout.
	result, err := h.daemon.procMgr.WaitReloadResult(60 * time.Second)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		return model.Response{
			OK: true,
			Data: model.ReloadResponse{
				Status:     "error",
				DurationMs: duration,
				Errors: []model.CompileError{
					{Message: err.Error()},
				},
				Warnings: []model.CompileError{},
			},
		}
	}

	status := "ok"
	var errors []model.CompileError
	if !result.Success {
		status = "error"
		if result.Error != "" {
			errors = append(errors, model.CompileError{Message: result.Error})
		}
	}

	return model.Response{
		OK: true,
		Data: model.ReloadResponse{
			Status:     status,
			DurationMs: result.DurationMs,
			Errors:     errors,
			Warnings:   []model.CompileError{},
		},
	}
}

func (h *Handler) handleRestart() model.Response {
	if h.daemon.procMgr == nil {
		return model.Response{
			OK:    false,
			Error: "flutter process not running",
		}
	}

	startTime := time.Now()

	if err := h.daemon.procMgr.SendRestart(); err != nil {
		return model.Response{
			OK:    false,
			Error: err.Error(),
		}
	}

	// Wait for restart result with 120s timeout (restart takes longer).
	result, err := h.daemon.procMgr.WaitReloadResult(120 * time.Second)
	if err != nil {
		duration := time.Since(startTime).Milliseconds()
		return model.Response{
			OK: true,
			Data: model.ReloadResponse{
				Status:     "error",
				DurationMs: duration,
				Errors: []model.CompileError{
					{Message: err.Error()},
				},
				Warnings: []model.CompileError{},
			},
		}
	}

	status := "ok"
	var errors []model.CompileError
	if !result.Success {
		status = "error"
		if result.Error != "" {
			errors = append(errors, model.CompileError{Message: result.Error})
		}
	}

	return model.Response{
		OK: true,
		Data: model.ReloadResponse{
			Status:     status,
			DurationMs: result.DurationMs,
			Errors:     errors,
			Warnings:   []model.CompileError{},
		},
	}
}

func (h *Handler) handleLogs(args map[string]any) model.Response {
	params := collector.QueryParams{
		Limit: 50,
	}

	// Parse query parameters from args.
	if v, ok := args["limit"]; ok {
		if n, ok := v.(float64); ok {
			params.Limit = int(n)
		}
	}
	if v, ok := args["since"]; ok {
		if s, ok := v.(string); ok {
			params.Since = s
		}
	}
	if v, ok := args["level"]; ok {
		if s, ok := v.(string); ok {
			params.Level = s
		}
	}
	if v, ok := args["source"]; ok {
		if s, ok := v.(string); ok {
			params.Source = s
		}
	}
	if v, ok := args["grep"]; ok {
		if s, ok := v.(string); ok {
			params.Grep = s
		}
	}
	if v, ok := args["all"]; ok {
		if b, ok := v.(bool); ok {
			params.All = b
		}
	}

	logs := h.daemon.QueryLogs(params)
	if logs == nil {
		logs = []model.LogEntry{}
	}

	return model.Response{
		OK: true,
		Data: model.LogsResponse{
			Count: len(logs),
			Logs:  logs,
		},
	}
}

func (h *Handler) handleAnalyze() model.Response {
	if h.daemon.project == "" {
		return model.Response{
			OK:    false,
			Error: "no project configured",
		}
	}

	result, err := analyzer.Run(h.daemon.project)
	if err != nil {
		return model.Response{
			OK:    false,
			Error: err.Error(),
		}
	}

	status := "ok"
	if len(result.Errors) > 0 {
		status = "error"
	}

	return model.Response{
		OK: true,
		Data: model.AnalyzeResponse{
			Status:     status,
			DurationMs: result.DurationMs,
			Errors:     result.Errors,
			Warnings:   result.Warnings,
			Infos:      result.Infos,
		},
	}
}

func (h *Handler) handleScreenshot() model.Response {
	debugURL := ""
	if h.daemon.procMgr != nil {
		debugURL = h.daemon.procMgr.DebugURL
	}

	if debugURL == "" {
		return model.Response{
			OK:    false,
			Error: "flutter app not running or no debug URL available",
		}
	}

	// Screenshots directory.
	screenshotDir := h.screenshotDir()

	s := snapshot.NewScreenshotter(
		h.daemon.project,
		h.daemon.device,
		debugURL,
		screenshotDir,
	)

	result, err := s.Capture()
	if err != nil {
		return model.Response{
			OK:    false,
			Error: fmt.Sprintf("screenshot failed: %v", err),
		}
	}

	return model.Response{
		OK: true,
		Data: model.ScreenshotResponse{
			Status: "ok",
			Path:   result.Path,
			Size:   result.Size,
			Device: result.Device,
		},
	}
}

func (h *Handler) handleInspect(args map[string]any) model.Response {
	debugURL := ""
	if h.daemon.procMgr != nil {
		debugURL = h.daemon.procMgr.DebugURL
	}

	if debugURL == "" {
		return model.Response{
			OK:    false,
			Error: "flutter app not running or no debug URL available",
		}
	}

	maxDepth := 0
	if v, ok := args["max_depth"]; ok {
		if n, ok := v.(float64); ok {
			maxDepth = int(n)
		}
	}

	respData, err := h.runInspectSubprocess(debugURL, maxDepth)
	if err != nil {
		return model.Response{OK: false, Error: fmt.Sprintf("inspect failed: %v", err)}
	}

	return model.Response{
		OK:   true,
		Data: respData,
	}
}

func (h *Handler) handleSemantics(args map[string]any) model.Response {
	payload, err := h.runInteraction("semantics", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

func (h *Handler) handleTap(args map[string]any) model.Response {
	payload, err := h.runInteraction("tap", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

func (h *Handler) handleType(args map[string]any) model.Response {
	payload, err := h.runInteraction("type", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

func (h *Handler) handleScroll(args map[string]any) model.Response {
	payload, err := h.runInteraction("scroll", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

func (h *Handler) handleSwipe(args map[string]any) model.Response {
	payload, err := h.runInteraction("swipe", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

func (h *Handler) handleLongPress(args map[string]any) model.Response {
	payload, err := h.runInteraction("longpress", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

func (h *Handler) handleWait(args map[string]any) model.Response {
	payload, err := h.runInteraction("wait", args)
	if err != nil {
		return model.Response{OK: false, Error: err.Error()}
	}
	return model.Response{OK: true, Data: payload}
}

// screenshotDir returns the directory for storing screenshots.
func (h *Handler) screenshotDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".flarness", "screenshots")
}

func (h *Handler) runInteraction(action string, args map[string]any) (map[string]any, error) {
	debugURL := ""
	if h.daemon.procMgr != nil {
		debugURL = h.daemon.procMgr.DebugURL
	}
	if debugURL == "" {
		return nil, fmt.Errorf("flutter app not running or no debug URL available")
	}

	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("encode args: %w", err)
	}

	cmd := exec.Command(exe, "_interact", "--debug-url", debugURL, "--action", action, "--args", string(argsJSON))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("decode interaction response: %w", err)
	}
	return payload, nil
}

func (h *Handler) runInspectSubprocess(debugURL string, maxDepth int) (map[string]any, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	cmd := exec.Command(exe, "_inspect", "--debug-url", debugURL, "--max-depth", fmt.Sprintf("%d", maxDepth))
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	var payload map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		return nil, fmt.Errorf("decode inspect response: %w", err)
	}
	return payload, nil
}

func stringValue(v any, fallback string) string {
	s, ok := v.(string)
	if !ok || s == "" {
		return fallback
	}
	return s
}
