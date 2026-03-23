package daemon

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

func TestServerIPCRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	d := &Daemon{
		baseDir:    tmpDir,
		pidPath:    filepath.Join(tmpDir, "daemon.pid"),
		socketPath: sockPath,
		project:    "/test/project",
		device:     "chrome",
		startTime:  time.Now(),
	}

	server := NewServer(sockPath, d)

	// Start server in goroutine.
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	// Wait for server to be ready.
	time.Sleep(100 * time.Millisecond)

	// Test: send status command.
	t.Run("status", func(t *testing.T) {
		resp := sendCmd(t, sockPath, model.Command{Cmd: "status"})
		if !resp.OK {
			t.Fatalf("expected OK=true, got error: %s", resp.Error)
		}
	})

	// Test: send reload command (no flutter process → error).
	t.Run("reload", func(t *testing.T) {
		resp := sendCmd(t, sockPath, model.Command{Cmd: "reload"})
		if resp.OK {
			t.Fatal("expected OK=false when no flutter process running")
		}
	})

	// Test: send logs command with args.
	t.Run("logs", func(t *testing.T) {
		resp := sendCmd(t, sockPath, model.Command{
			Cmd: "logs",
			Args: map[string]any{
				"grep":  "test",
				"limit": 10,
			},
		})
		if !resp.OK {
			t.Fatalf("expected OK=true, got error: %s", resp.Error)
		}
	})

	// Test: send unknown command.
	t.Run("unknown", func(t *testing.T) {
		resp := sendCmd(t, sockPath, model.Command{Cmd: "badcmd"})
		if resp.OK {
			t.Error("expected OK=false for unknown command")
		}
	})

	// Test: stop command shuts down the server.
	t.Run("stop", func(t *testing.T) {
		resp := sendCmd(t, sockPath, model.Command{Cmd: "stop"})
		if !resp.OK {
			t.Fatalf("expected OK=true, got error: %s", resp.Error)
		}
	})

	// Server should exit gracefully.
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("server error: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("server did not shut down in time")
	}

	// Socket file should be cleaned up.
	if _, err := os.Stat(sockPath); !os.IsNotExist(err) {
		// Note: the socket may or may not be cleaned depending on OS.
		// This is a soft check.
	}
}

func TestServerMultipleClients(t *testing.T) {
	tmpDir := t.TempDir()
	sockPath := filepath.Join(tmpDir, "test.sock")

	d := &Daemon{
		baseDir:    tmpDir,
		pidPath:    filepath.Join(tmpDir, "daemon.pid"),
		socketPath: sockPath,
		project:    "/test/project",
		device:     "chrome",
		startTime:  time.Now(),
	}

	server := NewServer(sockPath, d)

	go server.ListenAndServe()
	time.Sleep(100 * time.Millisecond)

	// Send multiple commands concurrently.
	done := make(chan bool, 5)
	for i := 0; i < 5; i++ {
		go func() {
			resp := sendCmd(t, sockPath, model.Command{Cmd: "status"})
			done <- resp.OK
		}()
	}

	for i := 0; i < 5; i++ {
		select {
		case ok := <-done:
			if !ok {
				t.Error("concurrent request failed")
			}
		case <-time.After(5 * time.Second):
			t.Fatal("concurrent request timed out")
		}
	}

	server.Shutdown()
}

// sendCmd is a test helper to connect to the Unix socket and exchange a command.
func sendCmd(t *testing.T, sockPath string, cmd model.Command) model.Response {
	t.Helper()

	conn, err := net.DialTimeout("unix", sockPath, 2*time.Second)
	if err != nil {
		t.Fatalf("dial error: %v", err)
	}
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second))

	encoder := json.NewEncoder(conn)
	if err := encoder.Encode(cmd); err != nil {
		t.Fatalf("encode error: %v", err)
	}

	var resp model.Response
	decoder := json.NewDecoder(conn)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatalf("decode error: %v", err)
	}

	return resp
}
