package cmd

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/canaanyjn/flarness/internal/daemon"
	"github.com/canaanyjn/flarness/internal/instance"
	"github.com/canaanyjn/flarness/internal/ipc"
	"github.com/canaanyjn/flarness/internal/model"
)

func TestWaitForStartedSessionReturnsWhenRunning(t *testing.T) {
	session := "abc12345"
	d, client, cleanup := testSessionHarness(t, session, []model.Response{
		{OK: true, Data: map[string]any{"flutter_state": "starting"}},
		{OK: true, Data: map[string]any{"flutter_state": "running", "url": "http://localhost:1234"}},
	})
	defer cleanup()

	status, err := waitForStartedSession(d, client)
	if err != nil {
		t.Fatalf("waitForStartedSession error: %v", err)
	}
	if status["flutter_state"] != "running" {
		t.Fatalf("flutter_state = %v, want running", status["flutter_state"])
	}
}

func TestWaitForStartedSessionFailsWhenStopped(t *testing.T) {
	session := "abc12345"
	d, client, cleanup := testSessionHarness(t, session, []model.Response{
		{OK: true, Data: map[string]any{"flutter_state": "stopped"}},
	})
	defer cleanup()

	_, err := waitForStartedSession(d, client)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestTailFileReturnsRecentLines(t *testing.T) {
	path := t.TempDir() + "/daemon.log"
	content := "a\nb\nc\nd\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile error: %v", err)
	}

	if got := tailFile(path, 2); got != "c\nd" {
		t.Fatalf("tailFile = %q, want %q", got, "c\nd")
	}
}

func testSessionHarness(t *testing.T, session string, responses []model.Response) (*daemon.Daemon, *ipc.Client, func()) {
	t.Helper()

	home := filepath.Join("/tmp", fmt.Sprintf("fh-home-%d", time.Now().UnixNano()))
	err := os.MkdirAll(home, 0755)
	if err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	t.Setenv("HOME", home)

	d := daemon.New(session)
	paths := instance.PathsForSession(session)
	if err := os.MkdirAll(paths.InstanceDir, 0755); err != nil {
		t.Fatalf("MkdirAll error: %v", err)
	}
	if err := d.WritePID(); err != nil {
		t.Fatalf("WritePID error: %v", err)
	}

	listener, err := net.Listen("unix", paths.SocketPath)
	if err != nil {
		t.Fatalf("listen error: %v", err)
	}

	go func() {
		idx := 0
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}

			var cmd model.Command
			_ = json.NewDecoder(conn).Decode(&cmd)

			resp := responses[len(responses)-1]
			if idx < len(responses) {
				resp = responses[idx]
			}
			idx++

			_ = json.NewEncoder(conn).Encode(resp)
			_ = conn.Close()
		}
	}()

	client := ipc.NewClient(session)
	cleanup := func() {
		_ = listener.Close()
		d.Cleanup()
		_ = os.RemoveAll(home)
	}
	return d, client, cleanup
}
