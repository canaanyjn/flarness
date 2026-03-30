package cliargs

import (
	"encoding/json"
	"strings"
)

// NormalizeExtraArgs accepts repeated CLI values as well as a single JSON array
// string such as ["--dart-define=FOO=bar"].
func NormalizeExtraArgs(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	var normalized []string
	for _, raw := range values {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}

		if strings.HasPrefix(raw, "[") {
			var parsed []string
			if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
				return nil, err
			}
			for _, item := range parsed {
				item = strings.TrimSpace(item)
				if item != "" {
					normalized = append(normalized, item)
				}
			}
			continue
		}

		normalized = append(normalized, raw)
	}

	if len(normalized) == 0 {
		return nil, nil
	}
	return normalized, nil
}
