package ipc

import (
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/canaanyjn/flarness/internal/instance"
	"github.com/canaanyjn/flarness/internal/model"
)

func TestClientSendReceive(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	// Start a mock server.
	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		// Read command.
		var cmd model.Command
		json.NewDecoder(conn).Decode(&cmd)

		// Send response.
		resp := model.Response{
			OK:   true,
			Data: map[string]any{"echo": cmd.Cmd},
		}
		json.NewEncoder(conn).Encode(resp)
	}()

	// Use client to send.
	client := &Client{session: "test", socketPath: sockPath}
	resp, err := client.Send(model.Command{Cmd: "test"})
	if err != nil {
		t.Fatalf("send error: %v", err)
	}

	if !resp.OK {
		t.Error("expected OK=true")
	}
}

func TestClientNotRunning(t *testing.T) {
	client := &Client{session: "test", socketPath: filepath.Join(t.TempDir(), "nonexistent.sock")}
	if client.IsRunning() {
		t.Error("expected IsRunning=false for nonexistent socket")
	}
}

func TestClientSendNoServer(t *testing.T) {
	client := &Client{session: "test", socketPath: filepath.Join(t.TempDir(), "nonexistent.sock")}
	_, err := client.Send(model.Command{Cmd: "test"})
	if err == nil {
		t.Error("expected error when no server is running")
	}
}

func TestNewClientUsesSessionPaths(t *testing.T) {
	client := NewClient("abc12345")
	paths := instance.PathsForSession("abc12345")
	if client.socketPath != paths.SocketPath {
		t.Errorf("socketPath = %q, want %q", client.socketPath, paths.SocketPath)
	}
}

func TestClientIsRunningWithServer(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	listener, err := net.Listen("unix", sockPath)
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}
	defer listener.Close()

	// Accept connections in background so IsRunning can connect.
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	time.Sleep(50 * time.Millisecond)

	client := &Client{session: "test", socketPath: sockPath}
	if !client.IsRunning() {
		t.Error("expected IsRunning=true when server is listening")
	}
}
