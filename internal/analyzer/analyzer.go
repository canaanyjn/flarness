package analyzer

import (
	"bufio"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

// Result holds the output of flutter analyze.
type Result struct {
	DurationMs int64
	Errors     []model.CompileError
	Warnings   []model.CompileError
	Infos      []model.CompileError
}

// Regex patterns for flutter analyze output.
var (
	// Matches: info • Unused import • lib/widgets/todo_item.dart:15:8 • unused_import
	analyzeLineRe = regexp.MustCompile(`^\s*(info|warning|error)\s+[•·-]\s+(.+?)\s+[•·-]\s+(.+?):(\d+):(\d+)\s+[•·-]\s+(\S+)\s*$`)

	// Matches: Analyzing project_name...
	analyzingRe = regexp.MustCompile(`^Analyzing\s+`)

	// Matches: X issues found. or No issues found!
	summaryRe = regexp.MustCompile(`^(\d+)\s+issue|^No issues found`)
)

// Run executes `flutter analyze` in the given project directory and parses the output.
func Run(projectDir string) (*Result, error) {
	start := time.Now()

	cmd := exec.Command("flutter", "analyze", "--no-pub")
	cmd.Dir = projectDir

	output, err := cmd.CombinedOutput()
	duration := time.Since(start).Milliseconds()

	result := &Result{
		DurationMs: duration,
		Errors:     []model.CompileError{},
		Warnings:   []model.CompileError{},
		Infos:      []model.CompileError{},
	}

	// Parse output line by line.
	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		parseLine(line, result)
	}

	// If flutter analyze exited with error but we got results, that's normal
	// (it exits with non-zero when issues are found).
	if err != nil && len(result.Errors) == 0 && len(result.Warnings) == 0 && len(result.Infos) == 0 {
		return nil, fmt.Errorf("flutter analyze failed: %w\nOutput: %s", err, string(output))
	}

	return result, nil
}

// parseLine parses a single line of flutter analyze output.
func parseLine(line string, result *Result) {
	matches := analyzeLineRe.FindStringSubmatch(line)
	if matches == nil {
		return
	}

	severity := matches[1]
	message := matches[2]
	file := matches[3]
	lineStr := matches[4]
	colStr := matches[5]
	code := matches[6]

	lineNum := parseInt(lineStr)
	colNum := parseInt(colStr)

	entry := model.CompileError{
		File:    file,
		Line:    lineNum,
		Col:     colNum,
		Message: message,
		Code:    code,
	}

	switch severity {
	case "error":
		result.Errors = append(result.Errors, entry)
	case "warning":
		result.Warnings = append(result.Warnings, entry)
	case "info":
		result.Infos = append(result.Infos, entry)
	}
}

// ParseOutput parses the combined output of flutter analyze (for testing).
func ParseOutput(output string) *Result {
	result := &Result{
		Errors:   []model.CompileError{},
		Warnings: []model.CompileError{},
		Infos:    []model.CompileError{},
	}

	scanner := bufio.NewScanner(strings.NewReader(output))
	for scanner.Scan() {
		parseLine(scanner.Text(), result)
	}

	return result
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		}
	}
	return n
}
