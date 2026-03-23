package process

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"time"
)

// State represents the current state of the Flutter process.
type State int

const (
	StateIdle     State = iota // Not started.
	StateStarting             // flutter run launched, waiting for app.started.
	StateRunning              // App is running.
	StateReloading            // Hot reload in progress.
	StateStopped              // Process exited.
)

func (s State) String() string {
	switch s {
	case StateIdle:
		return "idle"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateReloading:
		return "reloading"
	case StateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// EventCallback is called for each line of stdout or stderr from the flutter process.
type EventCallback func(source string, line string)

// Manager manages the flutter run --machine child process.
type Manager struct {
	mu sync.Mutex

	cmd     *exec.Cmd
	stdin   io.WriteCloser
	stdout  io.ReadCloser
	stderr  io.ReadCloser

	state   State
	project string
	device  string

	// Callback for each line of output (for parser to consume).
	onEvent EventCallback

	// Channel signaled when the process exits.
	doneCh chan struct{}

	// Debug port URL extracted from app.debugPort event.
	DebugURL string

	// App URL (e.g. http://localhost:8080).
	AppURL string

	// Reload tracking.
	reloadMu     sync.Mutex
	reloadResult chan ReloadResult
}

// ReloadResult holds the outcome of a hot reload or restart.
type ReloadResult struct {
	Success    bool
	DurationMs int64
	Error      string
}

// New creates a new process Manager.
func New(project, device string, onEvent EventCallback) *Manager {
	return &Manager{
		project: project,
		device:  device,
		state:   StateIdle,
		onEvent: onEvent,
		doneCh:  make(chan struct{}),
	}
}

// Start launches `flutter run --machine` as a child process.
func (m *Manager) Start(extraArgs []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateIdle && m.state != StateStopped {
		return fmt.Errorf("flutter process already in state: %s", m.state)
	}

	args := []string{"run", "--machine"}
	if m.device != "" {
		args = append(args, "-d", m.device)
	}
	args = append(args, extraArgs...)

	m.cmd = exec.Command("flutter", args...)
	m.cmd.Dir = m.project
	m.cmd.Env = append(os.Environ(), "FLUTTER_SUPPRESS_ANALYTICS=true")

	var err error

	m.stdin, err = m.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("stdin pipe: %w", err)
	}

	m.stdout, err = m.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}

	m.stderr, err = m.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := m.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start flutter: %w", err)
	}

	m.state = StateStarting
	m.doneCh = make(chan struct{})

	// Read stdout and stderr in separate goroutines.
	go m.readLines("stdout", m.stdout)
	go m.readLines("stderr", m.stderr)

	// Wait for process exit in background.
	go func() {
		m.cmd.Wait()
		m.mu.Lock()
		m.state = StateStopped
		m.mu.Unlock()
		close(m.doneCh)
	}()

	return nil
}

// readLines reads lines from a reader and forwards to the event callback.
func (m *Manager) readLines(source string, r io.Reader) {
	scanner := bufio.NewScanner(r)
	// Increase buffer for potentially large JSON lines.
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if m.onEvent != nil {
			m.onEvent(source, line)
		}
	}
}

// SendReload sends a hot reload command ("r") to flutter's stdin.
func (m *Manager) SendReload() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateRunning {
		return fmt.Errorf("cannot reload: process is %s", m.state)
	}

	m.state = StateReloading
	m.reloadResult = make(chan ReloadResult, 1)

	_, err := m.stdin.Write([]byte("r"))
	if err != nil {
		m.state = StateRunning
		return fmt.Errorf("failed to send reload: %w", err)
	}

	return nil
}

// SendRestart sends a hot restart command ("R") to flutter's stdin.
func (m *Manager) SendRestart() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state != StateRunning {
		return fmt.Errorf("cannot restart: process is %s", m.state)
	}

	m.state = StateReloading
	m.reloadResult = make(chan ReloadResult, 1)

	_, err := m.stdin.Write([]byte("R"))
	if err != nil {
		m.state = StateRunning
		return fmt.Errorf("failed to send restart: %w", err)
	}

	return nil
}

// WaitReloadResult waits for the reload/restart to complete with a timeout.
func (m *Manager) WaitReloadResult(timeout time.Duration) (ReloadResult, error) {
	m.reloadMu.Lock()
	ch := m.reloadResult
	m.reloadMu.Unlock()

	if ch == nil {
		return ReloadResult{}, fmt.Errorf("no reload in progress")
	}

	select {
	case result := <-ch:
		m.mu.Lock()
		m.state = StateRunning
		m.mu.Unlock()
		return result, nil
	case <-time.After(timeout):
		m.mu.Lock()
		m.state = StateRunning
		m.mu.Unlock()
		return ReloadResult{}, fmt.Errorf("reload timed out after %s", timeout)
	}
}

// NotifyReloadResult is called by the parser when a reload completes.
func (m *Manager) NotifyReloadResult(result ReloadResult) {
	m.reloadMu.Lock()
	ch := m.reloadResult
	m.reloadMu.Unlock()

	if ch != nil {
		select {
		case ch <- result:
		default:
		}
	}
}

// SetState updates the process state (called by the parser).
func (m *Manager) SetState(s State) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state = s
}

// GetState returns the current process state.
func (m *Manager) GetState() State {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state
}

// Stop sends "q" to flutter stdin to gracefully quit.
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state == StateIdle || m.state == StateStopped {
		return nil
	}

	if m.stdin != nil {
		m.stdin.Write([]byte("q"))
		m.stdin.Close()
	}

	// Wait for process to exit.
	select {
	case <-m.doneCh:
		// Process exited.
	case <-time.After(10 * time.Second):
		// Force kill.
		if m.cmd.Process != nil {
			m.cmd.Process.Kill()
		}
	}

	m.state = StateStopped
	return nil
}

// Done returns a channel that's closed when the process exits.
func (m *Manager) Done() <-chan struct{} {
	return m.doneCh
}
