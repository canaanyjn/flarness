package model

// Response represents an IPC response from Daemon to CLI.
type Response struct {
	OK    bool   `json:"ok"`
	Data  any    `json:"data,omitempty"`
	Error string `json:"error,omitempty"`
}

// StartResponse is the data payload for a successful start command.
type StartResponse struct {
	Status  string `json:"status"`
	PID     int    `json:"pid"`
	Device  string `json:"device"`
	URL     string `json:"url,omitempty"`
	Message string `json:"message,omitempty"`
}

// StopResponse is the data payload for a successful stop command.
type StopResponse struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// StatusResponse is the data payload for the status command.
type StatusResponse struct {
	Running bool      `json:"running"`
	PID     int       `json:"pid,omitempty"`
	Device  string    `json:"device,omitempty"`
	Project string    `json:"project,omitempty"`
	Uptime  string    `json:"uptime,omitempty"`
	URL     string    `json:"url,omitempty"`
	LogFile string    `json:"log_file,omitempty"`
	LogSize string    `json:"log_size,omitempty"`
	Stats   *LogStats `json:"stats,omitempty"`
}

// LogStats holds log statistics.
type LogStats struct {
	TotalLogs   int    `json:"total_logs"`
	Errors      int    `json:"errors"`
	Warnings    int    `json:"warnings"`
	LastReload  string `json:"last_reload,omitempty"`
	ReloadCount int    `json:"reload_count"`
}

// ReloadResponse is the data payload for reload/restart commands.
type ReloadResponse struct {
	Status     string         `json:"status"`
	DurationMs int64          `json:"duration_ms"`
	Errors     []CompileError `json:"errors"`
	Warnings   []CompileError `json:"warnings"`
}

// CompileError represents a single compilation error or warning.
type CompileError struct {
	File    string `json:"file"`
	Line    int    `json:"line"`
	Col     int    `json:"col,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// LogsResponse is the data payload for log queries.
type LogsResponse struct {
	Count int        `json:"count"`
	Logs  []LogEntry `json:"logs"`
}

// AnalyzeResponse is the data payload for the analyze command.
type AnalyzeResponse struct {
	Status     string         `json:"status"`
	DurationMs int64          `json:"duration_ms"`
	Errors     []CompileError `json:"errors"`
	Warnings   []CompileError `json:"warnings"`
	Infos      []CompileError `json:"infos"`
}

// ScreenshotResponse is the data payload for the screenshot command.
type ScreenshotResponse struct {
	Status string `json:"status"`
	Path   string `json:"path"`
	Size   string `json:"size"`
	Device string `json:"device"`
}

// InspectResponse is the data payload for the inspect command.
type InspectResponse struct {
	Status     string `json:"status"`
	WidgetTree any    `json:"widget_tree,omitempty"`
	RenderTree string `json:"render_tree,omitempty"`
	Summary    any    `json:"summary,omitempty"`
}

// SnapshotResponse is the data payload for the snapshot command (screenshot + inspect).
type SnapshotResponse struct {
	Status     string `json:"status"`
	Screenshot any    `json:"screenshot,omitempty"`
	WidgetTree any    `json:"widget_tree,omitempty"`
	RenderTree string `json:"render_tree,omitempty"`
	Summary    any    `json:"summary,omitempty"`
}

// SemanticsResponse is the data payload for the semantics command.
type SemanticsResponse struct {
	Status    string `json:"status"`
	Source    string `json:"source,omitempty"`
	Tree      any    `json:"tree,omitempty"`
	Summary   any    `json:"summary,omitempty"`
	Route     any    `json:"route,omitempty"`
	Extension string `json:"extension,omitempty"`
}

// InteractionResponse is the data payload for interaction commands.
type InteractionResponse struct {
	Status    string `json:"status"`
	Action    string `json:"action"`
	Message   string `json:"message,omitempty"`
	Matched   any    `json:"matched,omitempty"`
	Result    any    `json:"result,omitempty"`
	Extension string `json:"extension,omitempty"`
}
