package model

import (
	"encoding/json"
	"testing"
)

func TestCommandJSON(t *testing.T) {
	cmd := Command{
		Cmd: "logs",
		Args: map[string]any{
			"grep":  "overflow",
			"level": "error",
			"since": "5m",
			"limit": 50,
		},
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded Command
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Cmd != "logs" {
		t.Errorf("cmd: got %q, want %q", decoded.Cmd, "logs")
	}
	if decoded.Args["grep"] != "overflow" {
		t.Errorf("args.grep: got %v, want %q", decoded.Args["grep"], "overflow")
	}
	if decoded.Args["level"] != "error" {
		t.Errorf("args.level: got %v, want %q", decoded.Args["level"], "error")
	}
}

func TestCommandArgsOmitEmpty(t *testing.T) {
	cmd := Command{Cmd: "status"}

	data, err := json.Marshal(cmd)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	raw := string(data)
	// Args should be omitted when nil.
	if stringContains(raw, `"args"`) {
		t.Errorf("expected args to be omitted, got: %s", raw)
	}
}

func TestResponseJSON(t *testing.T) {
	tests := []struct {
		name string
		resp Response
		ok   bool
	}{
		{
			name: "success",
			resp: Response{
				OK: true,
				Data: StopResponse{
					Status:  "ok",
					Message: "daemon stopped",
				},
			},
			ok: true,
		},
		{
			name: "error",
			resp: Response{
				OK:    false,
				Error: "daemon not running",
			},
			ok: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.resp)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var decoded Response
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if decoded.OK != tt.ok {
				t.Errorf("ok: got %v, want %v", decoded.OK, tt.ok)
			}
		})
	}
}

func TestReloadResponseJSON(t *testing.T) {
	resp := ReloadResponse{
		Status:     "error",
		DurationMs: 150,
		Errors: []CompileError{
			{
				File:    "lib/pages/home_page.dart",
				Line:    42,
				Col:     10,
				Message: "Undefined name 'foo'",
				Code:    "undefined_identifier",
			},
		},
		Warnings: []CompileError{},
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded ReloadResponse
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Status != "error" {
		t.Errorf("status: got %q, want %q", decoded.Status, "error")
	}
	if decoded.DurationMs != 150 {
		t.Errorf("duration_ms: got %d, want 150", decoded.DurationMs)
	}
	if len(decoded.Errors) != 1 {
		t.Fatalf("errors count: got %d, want 1", len(decoded.Errors))
	}
	if decoded.Errors[0].File != "lib/pages/home_page.dart" {
		t.Errorf("error file: got %q", decoded.Errors[0].File)
	}
	if decoded.Errors[0].Line != 42 {
		t.Errorf("error line: got %d, want 42", decoded.Errors[0].Line)
	}
}
