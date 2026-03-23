package collector

import (
	"regexp"
	"strings"
	"time"

	"github.com/canaanyjn/flarness/internal/model"
)

// QueryParams defines filters for log queries.
type QueryParams struct {
	Limit  int      // Max entries to return (default: 50).
	Since  string   // Time filter: "30s", "5m", "1h".
	Level  string   // Comma-separated levels: "error,warning".
	Source string   // Comma-separated sources: "app,framework".
	Grep   string   // Regex pattern to search messages.
	All    bool     // Search all historical files, not just in-memory.
}

// Query runs a filtered query against the in-memory log buffer.
func (c *Collector) Query(params QueryParams) []model.LogEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Determine time cutoff.
	var cutoff time.Time
	if params.Since != "" {
		if d, err := parseDuration(params.Since); err == nil {
			cutoff = time.Now().Add(-d)
		}
	}

	// Parse level filter.
	var levels map[string]bool
	if params.Level != "" {
		levels = make(map[string]bool)
		for _, l := range strings.Split(params.Level, ",") {
			levels[strings.TrimSpace(l)] = true
		}
	}

	// Parse source filter.
	var sources map[string]bool
	if params.Source != "" {
		sources = make(map[string]bool)
		for _, s := range strings.Split(params.Source, ",") {
			sources[strings.TrimSpace(s)] = true
		}
	}

	// Compile regex.
	var grepRe *regexp.Regexp
	if params.Grep != "" {
		grepRe, _ = regexp.Compile(params.Grep)
	}

	// Set default limit.
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}

	// Filter entries (iterate backwards for most recent first).
	var result []model.LogEntry
	for i := len(c.buffer) - 1; i >= 0 && len(result) < limit; i-- {
		entry := c.buffer[i]

		// Time filter.
		if !cutoff.IsZero() && entry.Timestamp.Before(cutoff) {
			continue
		}

		// Level filter.
		if levels != nil && !levels[entry.Level] {
			continue
		}

		// Source filter.
		if sources != nil && !sources[entry.Source] {
			continue
		}

		// Grep filter.
		if grepRe != nil && !grepRe.MatchString(entry.Message) {
			continue
		}

		result = append(result, entry)
	}

	// Reverse to chronological order.
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result
}

// parseDuration parses a human-friendly duration string: "30s", "5m", "1h", "2d".
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return 0, nil
	}

	// Handle "d" suffix for days.
	if strings.HasSuffix(s, "d") {
		s = s[:len(s)-1] + "h"
		d, err := time.ParseDuration(s)
		if err != nil {
			return 0, err
		}
		return d * 24, nil
	}

	return time.ParseDuration(s)
}
