package parser

import (
	"testing"

	"github.com/canaanyjn/flarness/internal/model"
)

func TestStderrParserDartError(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("lib/pages/home_page.dart:42:10: Error: Undefined name 'foo'.")

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Level != model.LevelError {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelError)
	}
	if cb.logs[0].Source != model.SourceEngine {
		t.Errorf("source: got %q, want %q", cb.logs[0].Source, model.SourceEngine)
	}
	if cb.logs[0].File != "lib/pages/home_page.dart" {
		t.Errorf("file: got %q", cb.logs[0].File)
	}
	if cb.logs[0].Line != 42 {
		t.Errorf("line: got %d, want 42", cb.logs[0].Line)
	}
}

func TestStderrParserDartWarning(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("lib/widgets/todo_item.dart:15:1: Warning: Unused import.")

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Level != model.LevelWarning {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelWarning)
	}
}

func TestStderrParserGeneralError(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("Error: Compilation failed.")

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Level != model.LevelError {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelError)
	}
	if cb.logs[0].Message != "Compilation failed." {
		t.Errorf("message: got %q", cb.logs[0].Message)
	}
}

func TestStderrParserCompilerMessageHeader(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("Compiler message:")

	// Should be skipped — no logs generated.
	if len(cb.logs) != 0 {
		t.Errorf("expected 0 logs for compiler message header, got %d", len(cb.logs))
	}
}

func TestStderrParserPlainText(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("Running flutter build...")

	if len(cb.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(cb.logs))
	}
	if cb.logs[0].Level != model.LevelInfo {
		t.Errorf("level: got %q, want %q", cb.logs[0].Level, model.LevelInfo)
	}
	if cb.logs[0].Source != model.SourceEngine {
		t.Errorf("source: got %q, want %q", cb.logs[0].Source, model.SourceEngine)
	}
}

func TestStderrParserEmptyLine(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("")
	p.ParseLine("   ")

	if len(cb.logs) != 0 {
		t.Errorf("expected 0 logs for empty lines, got %d", len(cb.logs))
	}
}

func TestStderrParserDartErrorTriggersReloadFailure(t *testing.T) {
	cb := &mockCallback{}
	p := NewStderrParser(cb)

	p.ParseLine("lib/main.dart:10:5: Error: Expected ';' after this.")

	// Should also notify a reload failure.
	if len(cb.reloadResults) != 1 {
		t.Fatalf("expected 1 reload result, got %d", len(cb.reloadResults))
	}
	if cb.reloadResults[0].success {
		t.Error("expected reload success=false for compilation error")
	}
}

func TestFmtSscanf(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"42", 42},
		{"0", 0},
		{"10", 10},
		{"999", 999},
		{"", 0},
	}

	for _, tt := range tests {
		var got int
		fmt_sscanf(tt.input, &got)
		if got != tt.want {
			t.Errorf("fmt_sscanf(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
