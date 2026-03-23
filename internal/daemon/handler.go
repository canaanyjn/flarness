package daemon

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/canaanyjn/flarness/internal/analyzer"
	"github.com/canaanyjn/flarness/internal/collector"
	"github.com/canaanyjn/flarness/internal/inspector"
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
	case "snapshot":
		return h.handleSnapshot(cmd.Args)
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

	ins := inspector.NewInspector(debugURL)
	result, err := ins.Inspect()
	if err != nil {
		return model.Response{
			OK:    false,
			Error: fmt.Sprintf("inspect failed: %v", err),
		}
	}

	// Apply max_depth pruning if requested.
	maxDepth := 0
	if v, ok := args["max_depth"]; ok {
		if n, ok := v.(float64); ok {
			maxDepth = int(n)
		}
	}

	var widgetTree any
	if result.Tree != nil {
		if maxDepth > 0 {
			widgetTree = inspector.PruneTree(result.Tree, maxDepth)
		} else {
			widgetTree = result.Tree
		}
	}

	return model.Response{
		OK: true,
		Data: model.InspectResponse{
			Status:     "ok",
			WidgetTree: widgetTree,
			RenderTree: result.RenderTree,
			Summary:    result.Summary,
		},
	}
}

func (h *Handler) handleSnapshot(args map[string]any) model.Response {
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

	// Run screenshot and inspect concurrently.
	type screenshotResult struct {
		result *snapshot.ScreenshotResult
		err    error
	}
	type inspectResult struct {
		result *inspector.InspectResult
		err    error
	}

	var wg sync.WaitGroup
	var ssResult screenshotResult
	var insResult inspectResult

	screenshotDir := h.screenshotDir()

	wg.Add(2)

	go func() {
		defer wg.Done()
		s := snapshot.NewScreenshotter(
			h.daemon.project,
			h.daemon.device,
			debugURL,
			screenshotDir,
		)
		r, err := s.Capture()
		ssResult = screenshotResult{result: r, err: err}
	}()

	go func() {
		defer wg.Done()
		ins := inspector.NewInspector(debugURL)
		r, err := ins.Inspect()
		insResult = inspectResult{result: r, err: err}
	}()

	wg.Wait()

	// Build response even if one part failed.
	resp := model.SnapshotResponse{
		Status: "ok",
	}

	if ssResult.err != nil {
		resp.Status = "partial"
		fmt.Fprintf(os.Stderr, "[flarness] screenshot failed: %v\n", ssResult.err)
	} else if ssResult.result != nil {
		resp.Screenshot = model.ScreenshotResponse{
			Status: "ok",
			Path:   ssResult.result.Path,
			Size:   ssResult.result.Size,
			Device: ssResult.result.Device,
		}
	}

	if insResult.err != nil {
		if resp.Status == "partial" {
			resp.Status = "error"
		} else {
			resp.Status = "partial"
		}
		fmt.Fprintf(os.Stderr, "[flarness] inspect failed: %v\n", insResult.err)
	} else if insResult.result != nil {
		// Apply max_depth pruning if requested.
		maxDepth := 0
		if v, ok := args["max_depth"]; ok {
			if n, ok := v.(float64); ok {
				maxDepth = int(n)
			}
		}

		if insResult.result.Tree != nil {
			if maxDepth > 0 {
				resp.WidgetTree = inspector.PruneTree(insResult.result.Tree, maxDepth)
			} else {
				resp.WidgetTree = insResult.result.Tree
			}
		}
		resp.RenderTree = insResult.result.RenderTree
		resp.Summary = insResult.result.Summary
	}

	// Both failed.
	if ssResult.err != nil && insResult.err != nil {
		return model.Response{
			OK:    false,
			Error: fmt.Sprintf("screenshot: %v; inspect: %v", ssResult.err, insResult.err),
		}
	}

	return model.Response{
		OK:   true,
		Data: resp,
	}
}

// screenshotDir returns the directory for storing screenshots.
func (h *Handler) screenshotDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".flarness", "screenshots")
}
