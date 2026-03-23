package cdp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
	"github.com/gorilla/websocket"
)

func TestResolveURL(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"ws://127.0.0.1:8080/ws", "ws://127.0.0.1:8080/ws"},
		{"wss://127.0.0.1:8080/ws", "wss://127.0.0.1:8080/ws"},
		{"http://127.0.0.1:8080/ws", "ws://127.0.0.1:8080/ws"},
		{"https://127.0.0.1:8080/ws", "wss://127.0.0.1:8080/ws"},
	}

	for _, tt := range tests {
		b := NewBridge(tt.input, nil)
		got := b.resolveURL()
		if got != tt.want {
			t.Errorf("resolveURL(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestIsFlutterInternal(t *testing.T) {
	tests := []struct {
		msg  string
		want bool
	}{
		{"flutter: Hello", true},
		{"Restarted application in 2.3s.", true},
		{"Performing hot reload...", true},
		{"Hello World", false},
		{"User tapped button", false},
	}

	for _, tt := range tests {
		got := isFlutterInternal(tt.msg)
		if got != tt.want {
			t.Errorf("isFlutterInternal(%q) = %v, want %v", tt.msg, got, tt.want)
		}
	}
}

func TestHandleConsoleEvent(t *testing.T) {
	var captured []model.LogEntry
	b := NewBridge("ws://dummy", func(entry model.LogEntry) {
		captured = append(captured, entry)
	})

	tests := []struct {
		name    string
		params  consoleAPICalledParams
		wantLvl string
		wantMsg string
	}{
		{
			name: "log",
			params: consoleAPICalledParams{
				Type: "log",
				Args: []remoteObj{{Type: "string", Value: "hello world"}},
			},
			wantLvl: model.LevelInfo,
			wantMsg: "hello world",
		},
		{
			name: "error",
			params: consoleAPICalledParams{
				Type: "error",
				Args: []remoteObj{{Type: "string", Value: "something broke"}},
			},
			wantLvl: model.LevelError,
			wantMsg: "something broke",
		},
		{
			name: "warning",
			params: consoleAPICalledParams{
				Type: "warn",
				Args: []remoteObj{{Type: "string", Value: "deprecated API"}},
			},
			wantLvl: model.LevelWarning,
			wantMsg: "deprecated API",
		},
		{
			name: "debug",
			params: consoleAPICalledParams{
				Type: "debug",
				Args: []remoteObj{{Type: "string", Value: "debug info"}},
			},
			wantLvl: model.LevelDebug,
			wantMsg: "debug info",
		},
		{
			name: "description",
			params: consoleAPICalledParams{
				Type: "log",
				Args: []remoteObj{{Type: "object", Desc: "MyObject {a: 1}"}},
			},
			wantLvl: model.LevelInfo,
			wantMsg: "MyObject {a: 1}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captured = nil
			data, _ := json.Marshal(tt.params)
			b.handleConsoleEvent(data)

			if len(captured) != 1 {
				t.Fatalf("expected 1 log, got %d", len(captured))
			}
			if captured[0].Level != tt.wantLvl {
				t.Errorf("level: got %q, want %q", captured[0].Level, tt.wantLvl)
			}
			if captured[0].Message != tt.wantMsg {
				t.Errorf("message: got %q, want %q", captured[0].Message, tt.wantMsg)
			}
			if captured[0].Source != model.SourceApp {
				t.Errorf("source: got %q, want %q", captured[0].Source, model.SourceApp)
			}
		})
	}
}

func TestHandleConsoleEventSkipsFlutterInternal(t *testing.T) {
	var captured []model.LogEntry
	b := NewBridge("ws://dummy", func(entry model.LogEntry) {
		captured = append(captured, entry)
	})

	params := consoleAPICalledParams{
		Type: "log",
		Args: []remoteObj{{Type: "string", Value: "flutter: internal message"}},
	}
	data, _ := json.Marshal(params)
	b.handleConsoleEvent(data)

	if len(captured) != 0 {
		t.Errorf("expected 0 logs for flutter internal, got %d", len(captured))
	}
}

func TestHandleConsoleEventSkipsEmpty(t *testing.T) {
	var captured []model.LogEntry
	b := NewBridge("ws://dummy", func(entry model.LogEntry) {
		captured = append(captured, entry)
	})

	params := consoleAPICalledParams{
		Type: "log",
		Args: []remoteObj{},
	}
	data, _ := json.Marshal(params)
	b.handleConsoleEvent(data)

	if len(captured) != 0 {
		t.Errorf("expected 0 logs for empty args, got %d", len(captured))
	}
}

func TestBridgeConnectAndReceive(t *testing.T) {
	// Start a mock WebSocket server.
	upgrader := websocket.Upgrader{}
	var serverConn *websocket.Conn

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		serverConn, err = upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Logf("upgrade error: %v", err)
			return
		}

		// Read the Runtime.enable command.
		_, msg, err := serverConn.ReadMessage()
		if err != nil {
			return
		}

		var req cdpRequest
		json.Unmarshal(msg, &req)
		if req.Method != "Runtime.enable" {
			t.Errorf("expected Runtime.enable, got %s", req.Method)
		}

		// Send success response.
		serverConn.WriteJSON(map[string]any{"id": req.ID, "result": map[string]any{}})

		// Send a console event.
		event := cdpResponse{
			Method: "Runtime.consoleAPICalled",
		}
		params := consoleAPICalledParams{
			Type: "log",
			Args: []remoteObj{{Type: "string", Value: "test message from browser"}},
		}
		paramsData, _ := json.Marshal(params)
		event.Params = paramsData
		serverConn.WriteJSON(event)

		// Keep connection alive briefly.
		time.Sleep(500 * time.Millisecond)
		serverConn.Close()
	}))
	defer server.Close()

	// Convert http URL to ws URL.
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	var captured []model.LogEntry
	b := NewBridge(wsURL, func(entry model.LogEntry) {
		captured = append(captured, entry)
	})

	err := b.Connect(5 * time.Second)
	if err != nil {
		t.Fatalf("Connect error: %v", err)
	}
	defer b.Close()

	if !b.IsConnected() {
		t.Error("expected IsConnected=true")
	}

	// Wait for event to be received.
	time.Sleep(200 * time.Millisecond)

	if len(captured) != 1 {
		t.Fatalf("expected 1 captured log, got %d", len(captured))
	}
	if captured[0].Message != "test message from browser" {
		t.Errorf("message: got %q", captured[0].Message)
	}
}

func TestBridgeCloseIdempotent(t *testing.T) {
	b := NewBridge("ws://dummy", nil)

	// Close without connecting — should not panic.
	err := b.Close()
	if err != nil {
		t.Errorf("Close error: %v", err)
	}

	// Double close.
	err = b.Close()
	if err != nil {
		t.Errorf("second Close error: %v", err)
	}
}

func TestMultipleConsoleArgs(t *testing.T) {
	var captured []model.LogEntry
	b := NewBridge("ws://dummy", func(entry model.LogEntry) {
		captured = append(captured, entry)
	})

	params := consoleAPICalledParams{
		Type: "log",
		Args: []remoteObj{
			{Type: "string", Value: "User:"},
			{Type: "string", Value: "John"},
			{Type: "number", Value: 42},
		},
	}
	data, _ := json.Marshal(params)
	b.handleConsoleEvent(data)

	if len(captured) != 1 {
		t.Fatalf("expected 1 log, got %d", len(captured))
	}
	if captured[0].Message != "User: John 42" {
		t.Errorf("message: got %q, want %q", captured[0].Message, "User: John 42")
	}
}
