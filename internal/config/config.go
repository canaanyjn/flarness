package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds the global Flarness configuration.
type Config struct {
	Log      LogConfig                `yaml:"log"`
	Defaults DefaultConfig            `yaml:"defaults"`
	CDP      CDPConfig                `yaml:"cdp"`
	Projects map[string]ProjectConfig `yaml:"projects"`
}

// LogConfig holds log-related settings.
type LogConfig struct {
	MaxFileSize   string `yaml:"max_file_size"`  // e.g. "50MB"
	RetentionDays int    `yaml:"retention_days"` // default: 7
	BufferSize    int    `yaml:"buffer_size"`    // default: 1000
}

// DefaultConfig holds default run parameters.
type DefaultConfig struct {
	Device         string   `yaml:"device"` // default: "chrome"
	ExtraArgs      []string `yaml:"extra_args"`
	FlutterCommand []string `yaml:"flutter_command"`
}

// ProjectConfig holds a named project shortcut.
type ProjectConfig struct {
	Path           string   `yaml:"path"`
	Device         string   `yaml:"device,omitempty"`
	ExtraArgs      []string `yaml:"extra_args,omitempty"`
	FlutterCommand []string `yaml:"flutter_command,omitempty"`
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
		Projects: map[string]ProjectConfig{},
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
	if cfg.Projects == nil {
		cfg.Projects = map[string]ProjectConfig{}
	}
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

// LookupProject returns a named project config.
func (c Config) LookupProject(name string) (ProjectConfig, bool) {
	project, ok := c.Projects[name]
	return project, ok
}
