package parser

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

// MachineEvent represents a raw event from `flutter run --machine` stdout.
type MachineEvent struct {
	Event  string          `json:"event"`
	Params json.RawMessage `json:"params"`
}

// AppLogParams is the params for app.log events.
type AppLogParams struct {
	AppID string `json:"appId"`
	Log   string `json:"log"`
}

// AppStartParams is the params for app.start events.
type AppStartParams struct {
	AppID       string `json:"appId"`
	Directory   string `json:"directory"`
	DeviceID    string `json:"deviceId"`
	LaunchMode  string `json:"launchMode"`
	SupportsRestart bool `json:"supportsRestart"`
}

// AppDebugPortParams is the params for app.debugPort events.
type AppDebugPortParams struct {
	AppID    string `json:"appId"`
	Port     int    `json:"port"`
	BaseURI  string `json:"baseUri"`
	WSURI    string `json:"wsUri"`
}

// AppProgressParams is the params for app.progress events.
type AppProgressParams struct {
	AppID      string `json:"appId"`
	ID         string `json:"id"`
	Message    string `json:"message"`
	Finished   bool   `json:"finished"`
	ProgressID string `json:"progressId"`
}

// DaemonLogParams is the params for daemon.logMessage events.
type DaemonLogParams struct {
	Log   string `json:"log"`
	Error bool   `json:"error"`
}

// AppStartedParams is the params for app.started events.
type AppStartedParams struct {
	AppID string `json:"appId"`
}

// AppStopParams is the params for app.stop events.
type AppStopParams struct {
	AppID string `json:"appId"`
}

// Callback is the interface for receiving parsed events.
type Callback interface {
	OnLog(entry model.LogEntry)
	OnStateChange(event string, data map[string]string)
	OnReloadResult(success bool, durationMs int64, errMsg string)
}

// MachineParser parses the JSON event stream from `flutter run --machine`.
type MachineParser struct {
	callback Callback

	// Track reload timing.
	reloadStartTime time.Time
}

// NewMachineParser creates a new parser.
func NewMachineParser(cb Callback) *MachineParser {
	return &MachineParser{callback: cb}
}

// ParseLine parses a single line of stdout from the flutter process.
// Lines that are valid JSON with an "event" field are machine events.
// Other lines are treated as plain text logs.
func (p *MachineParser) ParseLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Try to parse as a machine event.
	if strings.HasPrefix(line, "[{") || strings.HasPrefix(line, "{") {
		// --machine wraps events in an array sometimes.
		cleaned := line
		if strings.HasPrefix(cleaned, "[") && strings.HasSuffix(cleaned, "]") {
			cleaned = cleaned[1 : len(cleaned)-1]
		}

		var event MachineEvent
		if err := json.Unmarshal([]byte(cleaned), &event); err == nil && event.Event != "" {
			p.handleEvent(event)
			return
		}
	}

	// Not a JSON event — treat as plain tool output.
	p.callback.OnLog(model.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     model.LevelInfo,
		Source:    model.SourceTool,
		Message:   line,
	})
}

// handleEvent dispatches a parsed machine event.
func (p *MachineParser) handleEvent(event MachineEvent) {
	switch event.Event {
	case "app.start":
		var params AppStartParams
		json.Unmarshal(event.Params, &params)
		p.callback.OnStateChange("app.start", map[string]string{
			"appId":    params.AppID,
			"deviceId": params.DeviceID,
		})

	case "app.started":
		p.callback.OnStateChange("app.started", nil)

	case "app.debugPort":
		var params AppDebugPortParams
		json.Unmarshal(event.Params, &params)
		p.callback.OnStateChange("app.debugPort", map[string]string{
			"wsUri":   params.WSURI,
			"baseUri": params.BaseURI,
		})

	case "app.log":
		var params AppLogParams
		if err := json.Unmarshal(event.Params, &params); err == nil {
			p.callback.OnLog(model.LogEntry{
				Timestamp: time.Now().UTC(),
				Level:     model.LevelInfo,
				Source:    model.SourceApp,
				Message:   strings.TrimSpace(params.Log),
			})
		}

	case "app.progress":
		var params AppProgressParams
		if err := json.Unmarshal(event.Params, &params); err == nil {
			if strings.Contains(params.ID, "hot.reload") || strings.Contains(params.ProgressID, "hot.reload") {
				if !params.Finished {
					p.reloadStartTime = time.Now()
				} else {
					duration := time.Since(p.reloadStartTime).Milliseconds()
					p.callback.OnReloadResult(true, duration, "")
				}
			}
			if strings.Contains(params.ID, "hot.restart") || strings.Contains(params.ProgressID, "hot.restart") {
				if !params.Finished {
					p.reloadStartTime = time.Now()
				} else {
					duration := time.Since(p.reloadStartTime).Milliseconds()
					p.callback.OnReloadResult(true, duration, "")
				}
			}

			// Log progress as tool message.
			if params.Message != "" {
				p.callback.OnLog(model.LogEntry{
					Timestamp: time.Now().UTC(),
					Level:     model.LevelInfo,
					Source:    model.SourceTool,
					Message:   params.Message,
				})
			}
		}

	case "app.stop":
		p.callback.OnStateChange("app.stop", nil)

	case "daemon.logMessage":
		var params DaemonLogParams
		if err := json.Unmarshal(event.Params, &params); err == nil {
			level := model.LevelInfo
			if params.Error {
				level = model.LevelError
			}
			// Detect framework errors (red screen, layout errors).
			source := model.SourceFramework
			msg := strings.TrimSpace(params.Log)

			if strings.Contains(msg, "Compiler message") || strings.Contains(msg, "Error:") {
				level = model.LevelError
			}

			p.callback.OnLog(model.LogEntry{
				Timestamp: time.Now().UTC(),
				Level:     level,
				Source:    source,
				Message:   msg,
			})
		}

	default:
		// Unknown event — log as debug.
		p.callback.OnLog(model.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     model.LevelDebug,
			Source:    model.SourceTool,
			Message:   "unknown event: " + event.Event,
		})
	}
}
