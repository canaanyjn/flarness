package ipc

import (
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/canaanyjn/flarness/internal/instance"
	"github.com/canaanyjn/flarness/internal/model"
)

const (
	connectTimeout = 5 * time.Second
	readTimeout    = 60 * time.Second
)

// Client communicates with the Flarness daemon over a Unix Domain Socket.
type Client struct {
	session    string
	socketPath string
}

// NewClient creates a new IPC client pointing to the daemon socket.
func NewClient(session string) *Client {
	paths := instance.PathsForSession(session)
	return &Client{
		session:    session,
		socketPath: paths.SocketPath,
	}
}

// Session returns the targeted daemon session id.
func (c *Client) Session() string {
	return c.session
}

// FlanressDir returns the base directory for Flarness data.
func FlanressDir() string {
	return instance.BaseDir()
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

// Send sends a command to the daemon and returns the parsed response.
func (c *Client) Send(cmd model.Command) (*model.Response, error) {
	conn, err := net.DialTimeout("unix", c.socketPath, connectTimeout)
	if err != nil {
		return nil, fmt.Errorf("cannot connect to daemon for session %s: %w", c.session, err)
	}
	defer conn.Close()

	conn.SetReadDeadline(time.Now().Add(readTimeout))

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		return nil, fmt.Errorf("failed to send command: %w", err)
	}

	decoder := json.NewDecoder(conn)
	var resp model.Response
	if err := decoder.Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	return &resp, nil
}
