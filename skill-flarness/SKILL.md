---
name: skill-flarness
description: Use Flarness to drive a Flutter project through an AI-friendly development loop with structured JSON output. Use when you need to start a Flutter app, inspect status, run hot reload or restart, capture screenshots or snapshots, inspect the widget tree, query logs, or run flutter analyze through the Flarness daemon.
---

# Flarness User

Use this skill when the goal is to operate a Flutter app through Flarness, not to maintain Flarness itself.

Bundled artifact:

- `bin/flarness-darwin-arm64`: bundled Flarness executable for macOS Apple Silicon, copied from this repository so the whole skill folder can be moved or reused directly.

## What Flarness does

Flarness is a CLI that wraps common Flutter development actions in structured JSON so an agent can drive the app reliably.

The important operating model is:

- Every CLI command should return structured JSON.
- A background daemon owns the running Flutter process.
- You usually start once, then issue follow-up commands against the daemon.
- The daemon keeps the Flutter VM service alive and exposes higher-level actions like reload, analyze, inspect, screenshot, snapshot, and logs.

## Preconditions

- Flutter must be installed and usable from the shell.
- The target directory must be a Flutter project with `pubspec.yaml`.
- The default device is `chrome` if no device is specified.
- Flarness stores runtime files under `~/.flarness`.

## Default workflow

1. Start Flarness against the Flutter project.
2. Check `status` until the daemon is up and the app URL or running state is available.
3. Make code changes in the Flutter app.
4. Run `reload` for incremental UI changes.
5. Use `snapshot` after reload when you need both visual state and widget-tree context.
6. Use `logs`, `errors`, or `analyze` when reload fails or the UI behaves incorrectly.
7. Use `restart` when app state is too stale for a hot reload to recover cleanly.
8. Use `stop` when you are done.

## Core commands

- `start`: starts the daemon and launches the Flutter app.
- `status`: returns whether the daemon is running and what project or device it controls.
- `reload`: sends hot reload.
- `restart`: sends hot restart.
- `screenshot`: captures the current app screen.
- `inspect`: returns a structured widget tree or render tree description.
- `snapshot`: returns screenshot plus widget-tree inspection together.
- `logs`: queries structured logs.
- `errors`: shortcut for error and fatal logs.
- `analyze`: runs `flutter analyze --no-pub` through Flarness.
- `stop`: stops the daemon.
- `help`: returns structured command help in JSON.

## Recommended command patterns

Start a project:

```bash
./bin/flarness-darwin-arm64 start --project /absolute/path/to/flutter_app --device chrome
```

If already inside the Flutter project:

```bash
./bin/flarness-darwin-arm64 start
```

Check state:

```bash
./bin/flarness-darwin-arm64 status
```

Reload after edits:

```bash
./bin/flarness-darwin-arm64 reload
```

Recover with restart:

```bash
./bin/flarness-darwin-arm64 restart
```

Get high-context UI state:

```bash
./bin/flarness-darwin-arm64 snapshot
```

Inspect widget tree only:

```bash
./bin/flarness-darwin-arm64 inspect --max-depth 6
```

Capture only a screenshot:

```bash
./bin/flarness-darwin-arm64 screenshot
```

Look for recent errors:

```bash
./bin/flarness-darwin-arm64 errors
./bin/flarness-darwin-arm64 logs --level error --since 5m
```

Run analyzer:

```bash
./bin/flarness-darwin-arm64 analyze
```

Stop the daemon:

```bash
./bin/flarness-darwin-arm64 stop
```

## How to use results

- Prefer parsing returned JSON instead of grepping plain text.
- `snapshot` is the best command after a UI change because it combines visual and structural context.
- Use `logs` for runtime failures, layout issues, and framework errors.
- Use `analyze` for static issues before or after reload.
- If `reload` returns an error payload, inspect the structured errors first, then fall back to `logs` or `restart`.

## Practical rules

- Start once per session instead of relaunching for every action.
- Default to `reload`; escalate to `restart` only when necessary.
- Use `snapshot` instead of separate `screenshot` and `inspect` calls when both are needed.
- If the daemon is not running, call `start` instead of retrying other commands.
- For web devices, screenshot may use CDP internally; otherwise Flarness falls back to Flutter's screenshot command.
- Keep the project path absolute when working across multiple repos to avoid ambiguity.

## Troubleshooting

- Error saying daemon is not running:
  run `./bin/flarness-darwin-arm64 start` in the Flutter project or pass `--project`.
- Error saying no `pubspec.yaml`:
  you are not pointing at a Flutter project root.
- Reload appears successful but UI is stale:
  run `./bin/flarness-darwin-arm64 restart`.
- Need machine-readable command schema:
  run `./bin/flarness-darwin-arm64 help` or `./bin/flarness-darwin-arm64 help <command>`.

## Good defaults for an agent using this skill

- Assume JSON output is the source of truth.
- Prefer `snapshot`, `logs`, and `analyze` over guessing what happened in the app.
- When a command fails, surface the structured error payload and choose the next command based on that result.
