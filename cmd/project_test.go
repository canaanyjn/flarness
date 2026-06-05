package cmd

import (
	"os"
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

func TestResolveProjectRootWalksUpToFlutterApp(t *testing.T) {
	root := t.TempDir()
	app := filepath.Join(root, "apps", "mobile")
	pkg := filepath.Join(app, "packages", "p2_core")
	deep := filepath.Join(pkg, "lib", "src")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(app, "lib"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(app, "ios"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The app is a runnable Flutter app (pubspec + lib/main.dart + ios/).
	mustWrite(t, filepath.Join(app, "pubspec.yaml"), "name: app\n")
	mustWrite(t, filepath.Join(app, "lib", "main.dart"), "void main() {}\n")
	// The nested package has a pubspec but no main.dart / platform dir.
	mustWrite(t, filepath.Join(pkg, "pubspec.yaml"), "name: p2_core\n")

	// From deep inside the nested package, we should resolve the app root,
	// not the package's own pubspec.yaml.
	if got := resolveProjectRoot(deep); got != app {
		t.Fatalf("resolveProjectRoot(deep) = %q, want %q", got, app)
	}
	// From the app root itself.
	if got := resolveProjectRoot(app); got != app {
		t.Fatalf("resolveProjectRoot(app) = %q, want %q", got, app)
	}
}

func TestResolveWrapperCommand(t *testing.T) {
	const project = "/work/worktrees/3e6b/p2/apps/mobile"
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{"relative wrapper resolves against project", []string{"scripts/dev.sh", "--flavor"},
			[]string{filepath.Join(project, "scripts/dev.sh"), "--flavor"}},
		{"bare command stays a PATH lookup", []string{"flutter"}, []string{"flutter"}},
		{"absolute path is untouched", []string{"/usr/local/bin/dev.sh"},
			[]string{"/usr/local/bin/dev.sh"}},
		{"empty stays empty", nil, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := resolveWrapperCommand(tc.in, project)
			if len(got) != len(tc.want) {
				t.Fatalf("len = %d, want %d (%v)", len(got), len(tc.want), got)
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
