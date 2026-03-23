package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the global Flarness configuration.
type Config struct {
	Log      LogConfig     `yaml:"log"`
	Defaults DefaultConfig `yaml:"defaults"`
	CDP      CDPConfig     `yaml:"cdp"`
}

// LogConfig holds log-related settings.
type LogConfig struct {
	MaxFileSize   string `yaml:"max_file_size"`   // e.g. "50MB"
	RetentionDays int    `yaml:"retention_days"`  // default: 7
	BufferSize    int    `yaml:"buffer_size"`     // default: 1000
}

// DefaultConfig holds default run parameters.
type DefaultConfig struct {
	Device    string   `yaml:"device"`      // default: "chrome"
	ExtraArgs []string `yaml:"extra_args"`
}

// CDPConfig holds CDP bridge settings.
type CDPConfig struct {
	Enabled bool   `yaml:"enabled"` // default: true
	Timeout string `yaml:"timeout"` // default: "10s"
}

// DefaultConfig returns the default configuration.
func Default() Config {
	return Config{
		Log: LogConfig{
			MaxFileSize:   "50MB",
			RetentionDays: 7,
			BufferSize:    1000,
		},
		Defaults: DefaultConfig{
			Device: "chrome",
		},
		CDP: CDPConfig{
			Enabled: true,
			Timeout: "10s",
		},
	}
}

// Load reads the config from ~/.flarness/config.yaml.
// Returns default config if file doesn't exist.
func Load() Config {
	cfg := Default()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg
	}

	path := filepath.Join(home, ".flarness", "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}

	yaml.Unmarshal(data, &cfg)
	return cfg
}

// Save writes the config to ~/.flarness/config.yaml.
func Save(cfg Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(home, ".flarness")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}

	path := filepath.Join(dir, "config.yaml")
	return os.WriteFile(path, data, 0644)
}
