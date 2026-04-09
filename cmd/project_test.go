package cmd

import (
	"path/filepath"
	"testing"

	"github.com/canaanyjn/flarness/internal/config"
)

func TestResolveProjectArgAlias(t *testing.T) {
	cfg := config.Default()
	cfg.Projects["p2-mobile"] = config.ProjectConfig{
		Path:   "/tmp/p2/apps/mobile",
		Device: "macos",
	}

	project, projectCfg, err := resolveProjectArg(cfg, "p2-mobile")
	if err != nil {
		t.Fatalf("resolveProjectArg error: %v", err)
	}
	if project != "/tmp/p2/apps/mobile" {
		t.Fatalf("project = %q, want /tmp/p2/apps/mobile", project)
	}
	if projectCfg.Device != "macos" {
		t.Fatalf("device = %q, want macos", projectCfg.Device)
	}
}

func TestResolveProjectArgPathFallback(t *testing.T) {
	cfg := config.Default()

	project, projectCfg, err := resolveProjectArg(cfg, "apps/mobile")
	if err != nil {
		t.Fatalf("resolveProjectArg error: %v", err)
	}
	want, _ := filepath.Abs("apps/mobile")
	if project != want {
		t.Fatalf("project = %q, want %q", project, want)
	}
	if projectCfg.Path != "" {
		t.Fatalf("expected empty project config, got %#v", projectCfg)
	}
}
