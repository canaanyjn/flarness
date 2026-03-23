package parser

import (
	"regexp"
	"strings"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

// StderrParser parses stderr output from the flutter process.
// Stderr typically contains compilation errors and dart analyzer output.
type StderrParser struct {
	callback Callback
}

// NewStderrParser creates a new stderr parser.
func NewStderrParser(cb Callback) *StderrParser {
	return &StderrParser{callback: cb}
}

// Regex patterns for parsing compilation errors.
var (
	// Matches: lib/main.dart:10:5: Error: Undefined name 'foo'.
	dartErrorRe = regexp.MustCompile(`^(.+\.dart):(\d+):(\d+):\s+(Error|Warning|Info):\s+(.+)$`)

	// Matches: Error: Compilation failed.
	generalErrorRe = regexp.MustCompile(`^Error:\s+(.+)$`)

	// Matches: Compiler message:
	compilerMessageRe = regexp.MustCompile(`^Compiler message:`)
)

// ParseLine parses a single line of stderr.
func (p *StderrParser) ParseLine(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}

	// Try Dart-style error format: file:line:col: Error/Warning: message
	if matches := dartErrorRe.FindStringSubmatch(line); matches != nil {
		level := model.LevelError
		switch strings.ToLower(matches[4]) {
		case "warning":
			level = model.LevelWarning
		case "info":
			level = model.LevelInfo
		}

		lineNum := 0
		colNum := 0
		fmt_sscanf(matches[2], &lineNum)
		fmt_sscanf(matches[3], &colNum)

		p.callback.OnLog(model.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     level,
			Source:    model.SourceEngine,
			Message:   matches[5],
			File:      matches[1],
			Line:      lineNum,
		})

		// Also signal as a reload error if we're in a reload cycle.
		if level == model.LevelError {
			p.callback.OnReloadResult(false, 0, matches[5])
		}
		return
	}

	// General error line.
	if matches := generalErrorRe.FindStringSubmatch(line); matches != nil {
		p.callback.OnLog(model.LogEntry{
			Timestamp: time.Now().UTC(),
			Level:     model.LevelError,
			Source:    model.SourceEngine,
			Message:   matches[1],
		})
		return
	}

	// Compiler message header — skip.
	if compilerMessageRe.MatchString(line) {
		return
	}

	// Default: log as engine info.
	p.callback.OnLog(model.LogEntry{
		Timestamp: time.Now().UTC(),
		Level:     model.LevelInfo,
		Source:    model.SourceEngine,
		Message:   line,
	})
}

// fmt_sscanf is a simple helper to parse an int from a string.
func fmt_sscanf(s string, v *int) {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	*v = n
}
