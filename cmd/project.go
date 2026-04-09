package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/canaanyjn/flarness/internal/config"
)

func resolveProjectArg(cfg config.Config, raw string) (string, config.ProjectConfig, error) {
	if raw == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", config.ProjectConfig{}, fmt.Errorf("cannot determine project path: %w", err)
		}
		project, _ := filepath.Abs(cwd)
		return project, config.ProjectConfig{}, nil
	}

	if project, ok := cfg.LookupProject(raw); ok {
		if project.Path == "" {
			return "", config.ProjectConfig{}, fmt.Errorf("project alias %q has no path configured", raw)
		}
		resolved, _ := filepath.Abs(project.Path)
		project.Path = resolved
		return resolved, project, nil
	}

	project, _ := filepath.Abs(raw)
	return project, config.ProjectConfig{}, nil
}
