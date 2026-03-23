package collector

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

// Collector is the central log collection and query engine.
// It maintains an in-memory ring buffer and persists logs to JSONL files.
type Collector struct {
	mu sync.RWMutex

	// In-memory ring buffer.
	buffer     []model.LogEntry
	bufferSize int

	// JSONL file for persistent storage.
	logDir      string
	logFile     *os.File
	logWriter   *bufio.Writer
	logFilePath string

	// Statistics.
	totalLogs   int
	errorCount  int
	warnCount   int
}

// Config holds configuration for the Collector.
type Config struct {
	LogDir     string // Directory for JSONL files.
	BufferSize int    // Max entries in memory (default: 1000).
}

// New creates a new Collector.
func New(cfg Config) (*Collector, error) {
	if cfg.BufferSize <= 0 {
		cfg.BufferSize = 1000
	}

	if err := os.MkdirAll(cfg.LogDir, 0755); err != nil {
		return nil, fmt.Errorf("cannot create log dir %s: %w", cfg.LogDir, err)
	}

	c := &Collector{
		buffer:     make([]model.LogEntry, 0, cfg.BufferSize),
		bufferSize: cfg.BufferSize,
		logDir:     cfg.LogDir,
	}

	// Open the current log file.
	if err := c.openLogFile(); err != nil {
		return nil, err
	}

	return c, nil
}

// openLogFile creates or opens the current.jsonl file for writing.
func (c *Collector) openLogFile() error {
	c.logFilePath = filepath.Join(c.logDir, "current.jsonl")
	f, err := os.OpenFile(c.logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("cannot open log file: %w", err)
	}
	c.logFile = f
	c.logWriter = bufio.NewWriter(f)
	return nil
}

// Add adds a log entry to both the in-memory buffer and the JSONL file.
func (c *Collector) Add(entry model.LogEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Add to ring buffer.
	if len(c.buffer) >= c.bufferSize {
		// Shift: remove oldest 10%.
		trim := c.bufferSize / 10
		if trim < 1 {
			trim = 1
		}
		c.buffer = c.buffer[trim:]
	}
	c.buffer = append(c.buffer, entry)

	// Update stats.
	c.totalLogs++
	switch entry.Level {
	case model.LevelError, model.LevelFatal:
		c.errorCount++
	case model.LevelWarning:
		c.warnCount++
	}

	// Persist to JSONL.
	if c.logWriter != nil {
		data, err := json.Marshal(entry)
		if err == nil {
			c.logWriter.Write(data)
			c.logWriter.WriteByte('\n')
		}
	}
}

// Flush writes buffered data to disk.
func (c *Collector) Flush() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logWriter != nil {
		return c.logWriter.Flush()
	}
	return nil
}

// Close flushes and closes the log file.
func (c *Collector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logWriter != nil {
		c.logWriter.Flush()
	}
	if c.logFile != nil {
		return c.logFile.Close()
	}
	return nil
}

// Archive renames current.jsonl to a timestamped file.
func (c *Collector) Archive() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logWriter != nil {
		c.logWriter.Flush()
	}
	if c.logFile != nil {
		c.logFile.Close()
	}

	ts := time.Now().Format("2006-01-02_15-04-05")
	archiveName := ts + ".jsonl"
	archivePath := filepath.Join(c.logDir, archiveName)

	if err := os.Rename(c.logFilePath, archivePath); err != nil {
		// If rename fails (e.g., no current file), that's ok.
		return "", nil
	}

	// Reopen a fresh current.jsonl.
	c.openLogFile()

	return archivePath, nil
}

// Stats returns current log statistics.
func (c *Collector) Stats() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return map[string]int{
		"total_logs": c.totalLogs,
		"errors":     c.errorCount,
		"warnings":   c.warnCount,
		"buffer":     len(c.buffer),
	}
}

// LogFilePath returns the path to the current log file.
func (c *Collector) LogFilePath() string {
	return c.logFilePath
}
