package daemon

import (
	"encoding/json"
	"testing"

	"github.com/canaanyjn/flarness/internal/model"
)

func TestHandlerRouting(t *testing.T) {
	d := &Daemon{
		project: "/test/project",
		device:  "chrome",
	}
	h := NewHandler(d)

	tests := []struct {
		cmd    string
		wantOK bool
	}{
		{"status", true},
		{"stop", true},
		{"reload", false},  // No procMgr → error.
		{"restart", false}, // No procMgr → error.
		{"logs", true},
		{"analyze", false}, // Fake project path → error.
		{"semantics", false},
		{"tap", false},
		{"type", false},
		{"wait", false},
		{"scroll", false},
		{"swipe", false},
		{"longpress", false},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.cmd, func(t *testing.T) {
			resp := h.Handle(model.Command{Cmd: tt.cmd})
			if resp.OK != tt.wantOK {
				t.Errorf("cmd=%q: OK got %v, want %v (error: %s)", tt.cmd, resp.OK, tt.wantOK, resp.Error)
			}
		})
	}
}

func TestHandlerStatusResponse(t *testing.T) {
	d := &Daemon{
		project: "/my/project",
		device:  "macOS",
	}
	h := NewHandler(d)

	resp := h.Handle(model.Command{Cmd: "status"})
	if !resp.OK {
		t.Fatalf("expected OK=true, got error: %s", resp.Error)
	}

	// Data should be a map with project info.
	data, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal data error: %v", err)
	}

	var status map[string]any
	if err := json.Unmarshal(data, &status); err != nil {
		t.Fatalf("unmarshal status error: %v", err)
	}

	if status["project"] != "/my/project" {
		t.Errorf("project: got %v, want /my/project", status["project"])
	}
	if status["device"] != "macOS" {
		t.Errorf("device: got %v, want macOS", status["device"])
	}
}

func TestHandlerReloadResponse(t *testing.T) {
	// Without procMgr, reload should return an error.
	d := &Daemon{}
	h := NewHandler(d)

	resp := h.Handle(model.Command{Cmd: "reload"})
	if resp.OK {
		t.Fatal("expected OK=false when no flutter process")
	}
	if resp.Error != "flutter process not running" {
		t.Errorf("error: got %q, want 'flutter process not running'", resp.Error)
	}
}

func TestHandlerRestartResponse(t *testing.T) {
	// Without procMgr, restart should return an error.
	d := &Daemon{}
	h := NewHandler(d)

	resp := h.Handle(model.Command{Cmd: "restart"})
	if resp.OK {
		t.Fatal("expected OK=false when no flutter process")
	}
	if resp.Error != "flutter process not running" {
		t.Errorf("error: got %q, want 'flutter process not running'", resp.Error)
	}
}

func TestHandlerLogsWithArgs(t *testing.T) {
	d := &Daemon{}
	h := NewHandler(d)

	resp := h.Handle(model.Command{
		Cmd: "logs",
		Args: map[string]any{
			"grep":  "overflow",
			"level": "error",
			"since": "5m",
			"limit": 50,
		},
	})

	if !resp.OK {
		t.Fatalf("expected OK=true, got error: %s", resp.Error)
	}

	data, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var logs model.LogsResponse
	if err := json.Unmarshal(data, &logs); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if logs.Count != 0 {
		t.Errorf("count: got %d, want 0 (P0 skeleton)", logs.Count)
	}
}

func TestHandlerUnknownCommand(t *testing.T) {
	d := &Daemon{}
	h := NewHandler(d)

	resp := h.Handle(model.Command{Cmd: "foobar"})
	if resp.OK {
		t.Error("expected OK=false for unknown command")
	}
	if resp.Error == "" {
		t.Error("expected non-empty error for unknown command")
	}
}
