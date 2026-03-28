package daemon

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/canaanyjn/flarness/internal/cdp"
	"github.com/canaanyjn/flarness/internal/collector"
	"github.com/canaanyjn/flarness/internal/model"
	"github.com/canaanyjn/flarness/internal/nativebridge"
	"github.com/canaanyjn/flarness/internal/parser"
	"github.com/canaanyjn/flarness/internal/platform"
	"github.com/canaanyjn/flarness/internal/process"
)

// Daemon manages the lifecycle of the Flarness background process.
type Daemon struct {
	baseDir    string
	pidPath    string
	socketPath string
	startTime  time.Time
	project    string
	device     string
	server     *Server

	// P1: Flutter process and parsers.
	procMgr       *process.Manager
	machineParser *parser.MachineParser
	stderrParser  *parser.StderrParser

	// P2: Log collector.
	collector *collector.Collector

	// P3: CDP bridge for Web platform.
	cdpBridge *cdp.Bridge

	// Native log bridge for Android.
	logcatBridge *nativebridge.LogcatBridge

	// Reload tracking.
	reloadCount int
	lastReload  time.Time
}

// New creates a new Daemon instance.
func New() *Daemon {
	home, _ := os.UserHomeDir()
	base := filepath.Join(home, ".flarness")
	return &Daemon{
		baseDir:    base,
		pidPath:    filepath.Join(base, "daemon.pid"),
		socketPath: filepath.Join(base, "daemon.sock"),
	}
}

// Start launches the daemon as a background process.
// If foreground is true, runs in the current process (used by the spawned daemon itself).
func (d *Daemon) Start(project, device string, extraArgs []string, foreground bool) error {
	// Ensure base directory exists.
	if err := os.MkdirAll(d.baseDir, 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", d.baseDir, err)
	}

	// Check if already running.
	if d.IsRunning() {
		pid, _ := d.ReadPID()
		return fmt.Errorf("daemon already running (pid: %d)", pid)
	}

	// Clean stale socket.
	os.Remove(d.socketPath)

	if foreground {
		return d.runForeground(project, device, extraArgs)
	}

	return d.spawnBackground(project, device, extraArgs)
}

// spawnBackground launches a new flarness process in daemon mode.
func (d *Daemon) spawnBackground(project, device string, extraArgs []string) error {
	// Resolve the current executable path.
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find executable: %w", err)
	}

	args := []string{"_daemon",
		"--project", project,
		"--device", device,
	}
	for _, a := range extraArgs {
		args = append(args, "--extra-args", a)
	}

	cmd := exec.Command(exe, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true, // Create a new session so it survives terminal close.
	}

	// Redirect stdout/stderr to log file for debugging.
	logFile, err := os.Create(filepath.Join(d.baseDir, "daemon.log"))
	if err == nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start daemon: %w", err)
	}

	// Wait briefly and check if the process is still alive.
	time.Sleep(500 * time.Millisecond)
	if cmd.Process == nil {
		return fmt.Errorf("daemon process failed to start")
	}

	// Don't wait for the child — let it run independently.
	cmd.Process.Release()

	return nil
}

// runForeground runs the daemon in the current process (called by _daemon subcommand).
func (d *Daemon) runForeground(project, device string, extraArgs []string) error {
	d.project = project
	d.device = device
	d.startTime = time.Now()

	// Write PID file.
	if err := d.WritePID(); err != nil {
		return fmt.Errorf("failed to write PID: %w", err)
	}
	defer d.Cleanup()

	// Write project metadata.
	d.WriteProjectMeta()

	// Initialize LogCollector.
	var err error
	d.collector, err = collector.New(collector.Config{
		LogDir:     d.LogDir(),
		BufferSize: 1000,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "[flarness] warning: log collector init failed: %v\n", err)
	}
	if d.collector != nil {
		defer d.collector.Close()
	}

	// Initialize parsers.
	d.machineParser = parser.NewMachineParser(d)
	d.stderrParser = parser.NewStderrParser(d)

	// Initialize and start the Flutter process.
	d.procMgr = process.New(project, device, d.onProcessEvent)
	if err := d.procMgr.Start(extraArgs); err != nil {
		fmt.Fprintf(os.Stderr, "[flarness] warning: flutter process failed to start: %v\n", err)
		// Continue without flutter — daemon still serves IPC for status/stop.
	}
	if platform.IsAndroid(device) && d.collector != nil {
		d.logcatBridge = nativebridge.NewLogcatBridge(device, d.OnLog)
		if err := d.logcatBridge.Start(); err != nil {
			fmt.Fprintf(os.Stderr, "[flarness] warning: logcat bridge failed to start: %v\n", err)
			d.logcatBridge = nil
		}
	}

	// Create and start the IPC server.
	d.server = NewServer(d.socketPath, d)

	fmt.Fprintf(os.Stderr, "[flarness] daemon started (pid=%d, project=%s, device=%s)\n",
		os.Getpid(), project, device)

	// Blocks until Stop() is called or signal received.
	return d.server.ListenAndServe()
}

// onProcessEvent is called for each line of stdout/stderr from flutter.
func (d *Daemon) onProcessEvent(source, line string) {
	switch source {
	case "stdout":
		d.machineParser.ParseLine(line)
	case "stderr":
		d.stderrParser.ParseLine(line)
	}
}

// --- parser.Callback interface implementation ---

// OnLog receives a parsed log entry from the parsers.
func (d *Daemon) OnLog(entry model.LogEntry) {
	if d.collector != nil {
		d.collector.Add(entry)
	}

	// Also print to daemon stderr for debugging.
	fmt.Fprintf(os.Stderr, "[%s][%s] %s\n", entry.Source, entry.Level, entry.Message)
}

// OnStateChange handles state transitions from the parser.
func (d *Daemon) OnStateChange(event string, data map[string]string) {
	switch event {
	case "app.started":
		if d.procMgr != nil {
			d.procMgr.SetState(process.StateRunning)
		}
		fmt.Fprintln(os.Stderr, "[flarness] Flutter app started")

	case "app.debugPort":
		if d.procMgr != nil && data != nil {
			d.procMgr.DebugURL = data["wsUri"]
			d.procMgr.AppURL = data["baseUri"]
		}
		fmt.Fprintf(os.Stderr, "[flarness] debug port: %s\n", data["wsUri"])

		// Auto-connect CDP bridge for Web platform.
		if d.isWebDevice() && data["wsUri"] != "" {
			go d.connectCDP(data["wsUri"])
		}

	case "app.stop":
		if d.procMgr != nil {
			d.procMgr.SetState(process.StateStopped)
		}
		if d.cdpBridge != nil {
			d.cdpBridge.Close()
		}
		if d.logcatBridge != nil {
			d.logcatBridge.Stop()
		}
		fmt.Fprintln(os.Stderr, "[flarness] Flutter app stopped")
	}
}

// isWebDevice checks if the current device is a web browser.
func (d *Daemon) isWebDevice() bool {
	device := strings.ToLower(d.device)
	return device == "chrome" || device == "web-server" || device == "edge" || strings.Contains(device, "web")
}

// connectCDP establishes the CDP bridge to capture browser console logs.
func (d *Daemon) connectCDP(wsURL string) {
	d.cdpBridge = cdp.NewBridge(wsURL, func(entry model.LogEntry) {
		d.OnLog(entry)
	})

	if err := d.cdpBridge.Connect(10 * time.Second); err != nil {
		fmt.Fprintf(os.Stderr, "[flarness] CDP connect failed: %v\n", err)
		d.cdpBridge = nil
		return
	}

	fmt.Fprintln(os.Stderr, "[flarness] CDP bridge connected (Web console logs enabled)")
}

// OnReloadResult handles reload completion events from the parser.
func (d *Daemon) OnReloadResult(success bool, durationMs int64, errMsg string) {
	d.reloadCount++
	d.lastReload = time.Now()

	if d.procMgr != nil {
		result := process.ReloadResult{
			Success:    success,
			DurationMs: durationMs,
			Error:      errMsg,
		}
		d.procMgr.NotifyReloadResult(result)
	}
}

// GetLogs returns logs from the collector with optional filtering.
func (d *Daemon) GetLogs() []model.LogEntry {
	if d.collector != nil {
		return d.collector.Query(collector.QueryParams{Limit: 1000})
	}
	return nil
}

// QueryLogs returns filtered logs from the collector.
func (d *Daemon) QueryLogs(params collector.QueryParams) []model.LogEntry {
	if d.collector != nil {
		return d.collector.Query(params)
	}
	return nil
}

// GetCollector returns the underlying collector (for handler use).
func (d *Daemon) GetCollector() *collector.Collector {
	return d.collector
}

// Stop gracefully shuts down the daemon.
func (d *Daemon) Stop() error {
	pid, err := d.ReadPID()
	if err != nil {
		return fmt.Errorf("daemon not running (no PID file)")
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		d.Cleanup()
		return fmt.Errorf("daemon process not found")
	}

	// Send SIGTERM for graceful shutdown.
	if err := proc.Signal(syscall.SIGTERM); err != nil {
		// Process might already be gone.
		d.Cleanup()
		return nil
	}

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if !processExists(proc) {
			d.Cleanup()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}

	// Force kill if graceful shutdown did not complete.
	if err := proc.Kill(); err == nil {
		killDeadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(killDeadline) {
			if !processExists(proc) {
				d.Cleanup()
				return nil
			}
			time.Sleep(100 * time.Millisecond)
		}
	}

	if processExists(proc) {
		return fmt.Errorf("daemon process did not exit after SIGTERM/SIGKILL")
	}

	d.Cleanup()
	return nil
}

func processExists(proc *os.Process) bool {
	if proc == nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// Cleanup removes PID file and socket.
func (d *Daemon) Cleanup() {
	if d.cdpBridge != nil {
		d.cdpBridge.Close()
	}
	if d.logcatBridge != nil {
		d.logcatBridge.Stop()
	}
	os.Remove(d.pidPath)
	os.Remove(d.socketPath)
}

// IsRunning checks if the daemon is currently running.
func (d *Daemon) IsRunning() bool {
	pid, err := d.ReadPID()
	if err != nil {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Check if process is alive.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// WritePID writes the current process PID to the PID file.
func (d *Daemon) WritePID() error {
	return os.WriteFile(d.pidPath, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// ReadPID reads the PID from the PID file.
func (d *Daemon) ReadPID() (int, error) {
	data, err := os.ReadFile(d.pidPath)
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(string(data))
}

// Status returns the current daemon status.
func (d *Daemon) Status() map[string]any {
	status := map[string]any{
		"running": d.project != "",
		"pid":     os.Getpid(),
		"device":  d.device,
		"project": d.project,
		"uptime":  time.Since(d.startTime).Round(time.Second).String(),
	}

	if d.collector != nil {
		stats := d.collector.Stats()
		status["log_file"] = d.collector.LogFilePath()
		status["log_size"] = d.collector.LogFileSize()
		status["stats"] = map[string]any{
			"total_logs":   stats["total_logs"],
			"errors":       stats["errors"],
			"warnings":     stats["warnings"],
			"reload_count": d.reloadCount,
		}
		if !d.lastReload.IsZero() {
			status["stats"].(map[string]any)["last_reload"] = d.lastReload.UTC().Format(time.RFC3339)
		}
	}

	return status
}

// ProjectMeta holds project metadata for storage.
type ProjectMeta struct {
	ProjectPath string `json:"project_path"`
	ProjectName string `json:"project_name"`
	Device      string `json:"device"`
	CreatedAt   string `json:"created_at"`
}

// WriteProjectMeta writes project metadata to the log directory.
func (d *Daemon) WriteProjectMeta() error {
	meta := ProjectMeta{
		ProjectPath: d.project,
		ProjectName: filepath.Base(d.project),
		Device:      d.device,
		CreatedAt:   time.Now().UTC().Format(time.RFC3339),
	}
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	metaDir := d.LogDir()
	if err := os.MkdirAll(metaDir, 0755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(metaDir, "meta.json"), data, 0644)
}

// LogDir returns the project-specific log directory.
func (d *Daemon) LogDir() string {
	// Use a simple hash of the project path for directory naming.
	h := fnvHash(d.project)
	return filepath.Join(d.baseDir, "logs", h)
}

// fnvHash returns an 8-char hex hash of a string (FNV-1a).
func fnvHash(s string) string {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}
