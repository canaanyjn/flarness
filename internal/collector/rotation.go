package collector

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// RotationConfig defines log rotation settings.
type RotationConfig struct {
	MaxFileSize    int64         // Max file size in bytes (default: 50MB).
	RetentionDays  int           // Days to keep old logs (default: 7).
}

// DefaultRotation returns default rotation settings.
func DefaultRotation() RotationConfig {
	return RotationConfig{
		MaxFileSize:   50 * 1024 * 1024, // 50MB
		RetentionDays: 7,
	}
}

// Rotate checks if the current log file needs rotation and rotates if needed.
func (c *Collector) Rotate(cfg RotationConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logFile == nil {
		return nil
	}

	info, err := c.logFile.Stat()
	if err != nil {
		return err
	}

	if info.Size() < cfg.MaxFileSize {
		return nil // No rotation needed.
	}

	// Flush and close.
	if c.logWriter != nil {
		c.logWriter.Flush()
	}
	c.logFile.Close()

	// Rename to timestamped file.
	ts := time.Now().Format("2006-01-02_15-04-05")
	archivePath := filepath.Join(c.logDir, ts+".jsonl")
	os.Rename(c.logFilePath, archivePath)

	// Open fresh file.
	return c.openLogFile()
}

// CleanOld removes log files older than retentionDays.
func (c *Collector) CleanOld(retentionDays int) (int, error) {
	entries, err := os.ReadDir(c.logDir)
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	removed := 0

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip current.jsonl and non-jsonl files.
		if name == "current.jsonl" || !strings.HasSuffix(name, ".jsonl") {
			continue
		}
		// Skip meta.json.
		if name == "meta.json" {
			continue
		}

		info, err := e.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			path := filepath.Join(c.logDir, name)
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

// ListFiles returns a list of all log files in the log directory.
func (c *Collector) ListFiles() ([]LogFileInfo, error) {
	entries, err := os.ReadDir(c.logDir)
	if err != nil {
		return nil, err
	}

	var files []LogFileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, LogFileInfo{
			Name:    e.Name(),
			Path:    filepath.Join(c.logDir, e.Name()),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}

	// Sort by modification time descending (newest first).
	sort.Slice(files, func(i, j int) bool {
		return files[i].ModTime.After(files[j].ModTime)
	})

	return files, nil
}

// LogFileInfo describes a log file.
type LogFileInfo struct {
	Name    string    `json:"name"`
	Path    string    `json:"path"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

// LogFileSize returns the current log file size as a human-readable string.
func (c *Collector) LogFileSize() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.logFile == nil {
		return "0B"
	}
	info, err := c.logFile.Stat()
	if err != nil {
		return "0B"
	}
	return humanSize(info.Size())
}

func humanSize(b int64) string {
	const unit = 1024
	if b < unit {
		return formatSize(b, "B")
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	suffixes := []string{"KB", "MB", "GB", "TB"}
	return formatSize(b/div, suffixes[exp])
}

func formatSize(n int64, suffix string) string {
	if n == 0 {
		return "0" + suffix
	}
	return strings.TrimRight(strings.TrimRight(
		strings.Replace(
			strings.Replace(
				formatInt(n), "", "", 0,
			), "", "", 0,
		), "0",
	), ".") + suffix
}

func formatInt(n int64) string {
	s := ""
	if n == 0 {
		return "0"
	}
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
