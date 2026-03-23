package model

import "time"

// LogEntry represents a single structured log entry.
type LogEntry struct {
	Timestamp  time.Time `json:"ts"`
	Level      string    `json:"level"`           // debug|info|warning|error|fatal
	Source     string    `json:"src"`             // app|framework|engine|tool
	Message    string    `json:"msg"`
	StackTrace string    `json:"stack,omitempty"`
	File       string    `json:"file,omitempty"`
	Line       int       `json:"line,omitempty"`
}

// LogLevel constants.
const (
	LevelDebug   = "debug"
	LevelInfo    = "info"
	LevelWarning = "warning"
	LevelError   = "error"
	LevelFatal   = "fatal"
)

// LogSource constants.
const (
	SourceApp       = "app"
	SourceFramework = "framework"
	SourceEngine    = "engine"
	SourceTool      = "tool"
)
