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
		return resolveProjectRoot(cwd), config.ProjectConfig{}, nil
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

// resolveProjectRoot walks up from start to the nearest ancestor that looks
// like a runnable Flutter project (a pubspec.yaml alongside lib/main.dart or a
// platform directory). This keeps a session keyed to the worktree's app root
// regardless of which subdirectory a command runs from, and avoids latching
// onto a nested package's pubspec.yaml in a monorepo. Falls back to the nearest
// pubspec.yaml, then to start itself.
func resolveProjectRoot(start string) string {
	abs, err := filepath.Abs(start)
	if err != nil {
		return start
	}

	for dir := abs; ; {
		if fileExists(filepath.Join(dir, "pubspec.yaml")) && isFlutterAppDir(dir) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	for dir := abs; ; {
		if fileExists(filepath.Join(dir, "pubspec.yaml")) {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return abs
}

// isFlutterAppDir reports whether dir looks like a runnable Flutter app rather
// than a plain Dart/Flutter package.
func isFlutterAppDir(dir string) bool {
	if fileExists(filepath.Join(dir, "lib", "main.dart")) {
		return true
	}
	for _, platform := range []string{"ios", "android", "macos", "web", "windows", "linux"} {
		if dirExists(filepath.Join(dir, platform)) {
			return true
		}
	}
	return false
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
