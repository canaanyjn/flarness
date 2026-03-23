package parser

import (
	"testing"

	"github.com/canaanyjn/flarness/internal/model"
)

// mockCallback captures events from the parser.
type mockCallback struct {
	logs         []model.LogEntry
	stateChanges []stateChange
	reloadResults []reloadResult
}

type stateChange struct {
	event string
	data  map[string]string
}

type reloadResult struct {
	success    bool
	durationMs int64
	errMsg     string
}

func (m *mockCallback) OnLog(entry model.LogEntry) {
	m.logs = append(m.logs, entry)
}

func (m *mockCallback) OnStateChange(event string, data map[string]string) {
	m.stateChanges = append(m.stateChanges, stateChange{event: event, data: data})
}

func (m *mockCallback) OnReloadResult(success bool, durationMs int64, errMsg string) {
	m.reloadResults = append(m.reloadResults, reloadResult{
		success: success, durationMs: durationMs, errMsg: errMsg,
	})
}

func TestMachineParserAppLog(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	// Simulate an app.log event from --machine stdout.
	p.ParseLine(`[{"event":"app.log","params":{"appId":"test","log":"Hello from Flutter"}}]`)

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Source != model.SourceApp {
		t.Errorf("source: got %q, want %q", cb.logs[0].Source, model.SourceApp)
	}
	if cb.logs[0].Message != "Hello from Flutter" {
		t.Errorf("message: got %q", cb.logs[0].Message)
	}
}

func TestMachineParserAppStarted(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine(`[{"event":"app.started","params":{"appId":"test"}}]`)

	if len(cb.stateChanges) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(cb.stateChanges))
	}
	if cb.stateChanges[0].event != "app.started" {
		t.Errorf("event: got %q, want app.started", cb.stateChanges[0].event)
	}
}

func TestMachineParserDebugPort(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine(`[{"event":"app.debugPort","params":{"appId":"test","port":8080,"baseUri":"http://localhost:8080","wsUri":"ws://127.0.0.1:8080/ws"}}]`)

	if len(cb.stateChanges) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(cb.stateChanges))
	}
	if cb.stateChanges[0].data["wsUri"] != "ws://127.0.0.1:8080/ws" {
		t.Errorf("wsUri: got %q", cb.stateChanges[0].data["wsUri"])
	}
}

func TestMachineParserDaemonLogMessage(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine(`[{"event":"daemon.logMessage","params":{"log":"RenderBox was not laid out","error":true}}]`)

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Source != model.SourceFramework {
		t.Errorf("source: got %q, want %q", cb.logs[0].Source, model.SourceFramework)
	}
	if cb.logs[0].Level != model.LevelError {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelError)
	}
}

func TestMachineParserDaemonLogMessageInfo(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine(`[{"event":"daemon.logMessage","params":{"log":"Some info message","error":false}}]`)

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Level != model.LevelInfo {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelInfo)
	}
}

func TestMachineParserAppStop(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine(`[{"event":"app.stop","params":{"appId":"test"}}]`)

	if len(cb.stateChanges) != 1 {
		t.Fatalf("expected 1 state change, got %d", len(cb.stateChanges))
	}
	if cb.stateChanges[0].event != "app.stop" {
		t.Errorf("event: got %q, want app.stop", cb.stateChanges[0].event)
	}
}

func TestMachineParserPlainText(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine("This is not JSON, just a plain tool message")

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Source != model.SourceTool {
		t.Errorf("source: got %q, want %q", cb.logs[0].Source, model.SourceTool)
	}
}

func TestMachineParserEmptyLine(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine("")
	p.ParseLine("   ")

	if len(cb.logs) != 0 {
		t.Errorf("expected 0 logs for empty lines, got %d", len(cb.logs))
	}
}

func TestMachineParserUnknownEvent(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	p.ParseLine(`[{"event":"some.unknown.event","params":{}}]`)

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log for unknown event, got %d", len(cb.logs))
	}
	if cb.logs[0].Level != model.LevelDebug {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelDebug)
	}
}

func TestMachineParserAppProgress(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	// Start a hot reload.
	p.ParseLine(`[{"event":"app.progress","params":{"appId":"test","id":"hot.reload","message":"Performing hot reload...","finished":false}}]`)

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 progress log, got %d", len(cb.logs))
	}

	// Finish the hot reload.
	p.ParseLine(`[{"event":"app.progress","params":{"appId":"test","id":"hot.reload","message":"","finished":true}}]`)

	if len(cb.reloadResults) != 1 {
		t.Fatalf("expected 1 reload result, got %d", len(cb.reloadResults))
	}
	if !cb.reloadResults[0].success {
		t.Error("expected reload success=true")
	}
}

func TestMachineParserNonArrayJSON(t *testing.T) {
	cb := &mockCallback{}
	p := NewMachineParser(cb)

	// Some flutter versions output without the wrapping array.
	p.ParseLine(`{"event":"app.log","params":{"appId":"test","log":"Direct JSON"}}`)

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Message != "Direct JSON" {
		t.Errorf("message: got %q", cb.logs[0].Message)
	}
}
