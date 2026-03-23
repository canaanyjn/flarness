package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

const (
	defaultSocketName = "daemon.sock"
	connectTimeout    = 5 * time.Second
	readTimeout       = 60 * time.Second
)

// Client communicates with the Flarness daemon over a Unix Domain Socket.
type Client struct {
	socketPath string
}

// NewClient creates a new IPC client pointing to the daemon socket.
func NewClient() *Client {
	return &Client{
		socketPath: SocketPath(),
	}
}

// SocketPath returns the default daemon socket path.
func SocketPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".flarness", defaultSocketName)
}

// FlanressDir returns the base directory for Flarness data.
func FlanressDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".flarness")
}

// Send sends a command to the daemon and returns the parsed response.
func (c *Client) Send(cmd model.Command) (*model.Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon (is it running?): %w", err)
	}
	defer conn.Close()

	// Set read deadline.
	conn.SetReadDeadline(time.Now().Add(readTimeout))

	// Encode and send the command.
	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	// Read response.
	decoder := json.NewDecoder(conn)
	var resp model.Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}

// IsRunning checks if the daemon socket exists and is connectable.
func (c *Client) IsRunning() bool {
	conn, err := net.DialTimeout("unix", c.socketPath, 1*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// PIDPath returns the path to the daemon PID file.
func PIDPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".flarness", "daemon.pid")
}
