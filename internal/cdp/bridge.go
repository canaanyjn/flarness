package cdp

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
	"github.com/gorilla/websocket"
)

// LogCallback is called for each console log captured from the browser.
type LogCallback func(entry model.LogEntry)

// Bridge connects to Chrome DevTools Protocol to capture browser console logs.
// This supplements the --machine stdout for Web platform where print() output
// appears in the browser console rather than the terminal.
type Bridge struct {
	mu       sync.Mutex
	conn     *websocket.Conn
	wsURL    string
	callback LogCallback
	done     chan struct{}
	closed   bool
}

// NewBridge creates a new CDP bridge.
func NewBridge(wsURL string, callback LogCallback) *Bridge {
	return &Bridge{
		wsURL:    wsURL,
		callback: callback,
		done:     make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection to Chrome DevTools.
func (b *Bridge) Connect(timeout time.Duration) error {
	// The wsURL from flutter may be a VM service URL.
	// We need to transform it for CDP if needed.
	targetURL := b.resolveURL()

	dialer := websocket.Dialer{
		HandshakeTimeout: timeout,
	}

	conn, _, err := dialer.Dial(targetURL, nil)
	if err != nil {
		return fmt.Errorf("CDP connect failed: %w", err)
	}

	b.mu.Lock()
	b.conn = conn
	b.mu.Unlock()

	// Enable Runtime domain to receive console events.
	if err := b.sendCommand("Runtime.enable", nil); err != nil {
		conn.Close()
		return fmt.Errorf("Runtime.enable failed: %w", err)
	}

	// Start listening for events.
	go b.readLoop()

	return nil
}

// resolveURL transforms the debug URL for CDP connection.
func (b *Bridge) resolveURL() string {
	// If it's already a ws:// or wss:// URL, use as-is.
	if strings.HasPrefix(b.wsURL, "ws://") || strings.HasPrefix(b.wsURL, "wss://") {
		return b.wsURL
	}

	// Try to parse and fix.
	u, err := url.Parse(b.wsURL)
	if err != nil {
		return b.wsURL
	}

	if u.Scheme == "http" {
		u.Scheme = "ws"
	} else if u.Scheme == "https" {
		u.Scheme = "wss"
	}

	return u.String()
}

// cdpRequest is a CDP JSON-RPC request.
type cdpRequest struct {
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params any    `json:"params,omitempty"`
}

// cdpResponse is a CDP JSON-RPC response/event.
type cdpResponse struct {
	ID     int             `json:"id,omitempty"`
	Method string          `json:"method,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
	Error  *cdpError       `json:"error,omitempty"`
}

type cdpError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// consoleAPICalledParams represents Runtime.consoleAPICalled event params.
type consoleAPICalledParams struct {
	Type string      `json:"type"` // log, warn, error, info, debug
	Args []remoteObj `json:"args"`
}

// remoteObj is a simplified CDP RemoteObject.
type remoteObj struct {
	Type  string `json:"type"`
	Value any    `json:"value"`
	Desc  string `json:"description,omitempty"`
}

var requestID int

// sendCommand sends a CDP command.
func (b *Bridge) sendCommand(method string, params any) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn == nil {
		return fmt.Errorf("not connected")
	}

	requestID++
	req := cdpRequest{
		ID:     requestID,
		Method: method,
		Params: params,
	}

	return b.conn.WriteJSON(req)
}

// readLoop reads CDP events from the WebSocket connection.
func (b *Bridge) readLoop() {
	defer func() {
		b.mu.Lock()
		b.closed = true
		b.mu.Unlock()
		close(b.done)
	}()

	for {
		b.mu.Lock()
		conn := b.conn
		b.mu.Unlock()

		if conn == nil {
			return
		}

		_, message, err := conn.ReadMessage()
		if err != nil {
			// Connection closed or error.
			return
		}

		var resp cdpResponse
		if err := json.Unmarshal(message, &resp); err != nil {
			continue
		}

		// Handle Runtime.consoleAPICalled events.
		if resp.Method == "Runtime.consoleAPICalled" {
			b.handleConsoleEvent(resp.Params)
		}
	}
}

// handleConsoleEvent processes a Runtime.consoleAPICalled event.
func (b *Bridge) handleConsoleEvent(params json.RawMessage) {
	var p consoleAPICalledParams
	if err := json.Unmarshal(params, &p); err != nil {
		return
	}

	// Map console type to log level.
	level := model.LevelInfo
	switch p.Type {
	case "error":
		level = model.LevelError
	case "warn", "warning":
		level = model.LevelWarning
	case "debug":
		level = model.LevelDebug
	case "info", "log":
		level = model.LevelInfo
	}

	// Extract message from args.
	var parts []string
	for _, arg := range p.Args {
		if arg.Desc != "" {
			parts = append(parts, arg.Desc)
		} else if arg.Value != nil {
			parts = append(parts, fmt.Sprintf("%v", arg.Value))
		}
	}

	msg := strings.Join(parts, " ")
	if msg == "" {
		return
	}

	// Skip flutter-internal messages that are already captured via --machine.
	if isFlutterInternal(msg) {
		return
	}

	entry := model.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     level,
		Source:    model.SourceApp,
		Message:   msg,
	}

	if b.callback != nil {
		b.callback(entry)
	}
}

// isFlutterInternal checks if a message is from Flutter internals
// (already captured via --machine stdout).
func isFlutterInternal(msg string) bool {
	internals := []string{
		"flutter: ",
		"Restarted application",
		"Performing hot reload",
		"Performing hot restart",
	}
	for _, prefix := range internals {
		if strings.HasPrefix(msg, prefix) {
			return true
		}
	}
	return false
}

// Close shuts down the CDP bridge.
func (b *Bridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.conn != nil && !b.closed {
		err := b.conn.Close()
		b.conn = nil
		return err
	}
	return nil
}

// Done returns a channel that's closed when the bridge disconnects.
func (b *Bridge) Done() <-chan struct{} {
	return b.done
}

// IsConnected returns whether the CDP bridge is currently connected.
func (b *Bridge) IsConnected() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.conn != nil && !b.closed
}
