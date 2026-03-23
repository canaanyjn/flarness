package ipc

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

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
	client := &Client{socketPath: sockPath}
	resp, err := client.Send(model.Command{Cmd: "test"})
	if err != nil {
		t.Fatalf("send error: %v", err)
	}

	if !resp.OK {
		t.Error("expected OK=true")
	}
}

func TestClientNotRunning(t *testing.T) {
	client := &Client{socketPath: filepath.Join(t.TempDir(), "nonexistent.sock")}
	if client.IsRunning() {
		t.Error("expected IsRunning=false for nonexistent socket")
	}
}

func TestClientSendNoServer(t *testing.T) {
	client := &Client{socketPath: filepath.Join(t.TempDir(), "nonexistent.sock")}
	_, err := client.Send(model.Command{Cmd: "test"})
	if err == nil {
		t.Error("expected error when no server is running")
	}
}

func TestSocketPath(t *testing.T) {
	path := SocketPath()
	if path == "" {
		t.Fatal("SocketPath should not be empty")
	}

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".flarness", "daemon.sock")
	if path != expected {
		t.Errorf("SocketPath: got %q, want %q", path, expected)
	}
}

func TestPIDPath(t *testing.T) {
	path := PIDPath()
	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".flarness", "daemon.pid")
	if path != expected {
		t.Errorf("PIDPath: got %q, want %q", path, expected)
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

	client := &Client{socketPath: sockPath}
	if !client.IsRunning() {
		t.Error("expected IsRunning=true when server is listening")
	}
}
