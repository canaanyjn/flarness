---
name: skill-flarness
description: Use Flarness to drive a Flutter project through an AI-friendly development loop with structured JSON output. Use when you need to start a Flutter app, inspect status, run hot reload or restart, capture screenshots, inspect the widget tree, query logs, run interaction subcommands, or run flutter analyze through the Flarness daemon.
---

# skill-flarness

Use this skill when the goal is to operate a Flutter app through Flarness, not to maintain Flarness itself.

## What Flarness does

Flarness is a CLI that wraps common Flutter development actions in structured JSON so an agent can drive the app reliably.

The important operating model is:

- Every CLI command should return structured JSON.
- A background daemon owns the running Flutter process.
- You usually start once, then issue follow-up commands against the daemon.
- The daemon keeps the Flutter VM service alive and exposes higher-level actions like reload, analyze, inspect, screenshot, logs, and grouped interaction commands.

## Preconditions

- `flarness` should be available on `PATH`.
- Flutter must be installed and usable from the shell.
- The target directory must be a Flutter project with `pubspec.yaml`.
- The default device is `chrome` if no device is specified.
- Flarness stores runtime files under `~/.flarness`.

## Installing Flarness

- If `flarness` is missing, install it before trying to operate the app.
- For Darwin/Linux hosts, prefer the published installer:

```bash
curl -fsSL https://raw.githubusercontent.com/canaanyjn/flarness/main/release/install.sh | sh
```

- To install a specific release into a user-writable directory:

```bash
curl -fsSL https://raw.githubusercontent.com/canaanyjn/flarness/main/release/install.sh | \
  RELEASE_VERSION=v0.1.0 INSTALL_DIR="$HOME/.local/bin" sh
```

- If the environment is already inside the Flarness repo, building from source is also acceptable:

```bash
make build
sudo make install
```

- After installation, verify with:

```bash
flarness help
```

## Default workflow

1. Start Flarness against the Flutter project.
2. Check `status` until the daemon is up and the app URL or running state is available.
3. Make code changes in the Flutter app.
4. Run `reload` for incremental UI changes.
5. Use `screenshot` and `inspect` separately after reload when you need both visual state and widget-tree context.
6. Use `logs` or `analyze` when reload fails or the UI behaves incorrectly.
7. Use `restart` when app state is too stale for a hot reload to recover cleanly.
8. Use `stop` when you are done.

## Core commands

- `start`: starts the daemon and launches the Flutter app.
- `status`: returns whether the daemon is running and what project or device it controls.
- `reload`: sends hot reload.
- `restart`: sends hot restart.
- `screenshot`: captures the current app screen.
- `inspect`: returns the structural debugging view, using widget tree or render tree data.
- `semantics`: returns the automation-facing view with labels, actions, focus state, and bounds.
- `logs`: queries structured logs.
- `analyze`: runs `flutter analyze --no-pub` through Flarness.
- `stop`: stops the daemon.
- `help`: returns structured command help in JSON.

## Choosing the right view

- Use `semantics` when the goal is automation: locating elements, checking labels, checking available actions, and driving `interact` subcommands.
- Use `inspect` when the goal is structural debugging: understanding widget composition, render hierarchy, or why the UI is laid out a certain way.
- If `inspect` and `semantics` seem to disagree, trust `semantics` for interaction decisions and trust `inspect` for structure/debugging decisions.
- For UI automation, the usual priority is: `semantics` first, `inspect` second.

## Recommended command patterns

Start a project:

```bash
flarness start --project /absolute/path/to/flutter_app --device chrome
```

If already inside the Flutter project:

```bash
flarness start
```

Check state:

```bash
flarness status
```

Reload after edits:

```bash
flarness reload
```

Recover with restart:

```bash
flarness restart
```

Inspect widget tree only:

```bash
flarness inspect --max-depth 6
```

Dump the automation-facing semantics tree:

```bash
flarness semantics
```

Capture only a screenshot:

```bash
flarness screenshot
```

Look for recent errors:

```bash
flarness logs --level error --since 5m
```

Run analyzer:

```bash
flarness analyze
```

Stop the daemon:

```bash
flarness stop
```

## Recommended interaction sequence

- Start with `flarness semantics` to see what the UI exposes for automation.
- Use `flarness interact tap` to focus or select the target element.
- Use `flarness interact type` only after focus is confirmed or intentionally set.
- Use `flarness interact wait` when the next UI state is expected to appear asynchronously.
- After every write or navigation action, run `flarness semantics` again to verify the UI actually changed.
- Use `flarness inspect` only when interaction succeeds but the structure or layout still needs explanation.

## How to use results

- Prefer parsing returned JSON instead of grepping plain text.
- Use `screenshot` for visual state, `inspect` for structural debugging, and `semantics` for interaction targeting after a UI change.
- Use `logs` for runtime failures, layout issues, and framework errors.
- Use `analyze` for static issues before or after reload.
- If `reload` returns an error payload, inspect the structured errors first, then fall back to `logs` or `restart`.

## Verifying success

- Do not treat `status: ok` from an interaction command as sufficient proof that the UI changed.
- After create/update actions, verify via `semantics`, status banners, or visible button-label changes.
- After state transitions, confirm the action label changed as expected, for example `Start` to `Complete` or `Complete` to `Reopen`.
- After text entry, confirm the expected value appears in the focused field or in the resulting status message.

## Practical rules

- Start once per session instead of relaunching for every action.
- Default to `reload`; escalate to `restart` only when necessary.
- Keep commands atomic: call `screenshot` and `inspect` separately when both are needed.
- If the daemon is not running, call `start` instead of retrying other commands.
- For web devices, screenshot may use CDP internally; otherwise Flarness falls back to Flutter's screenshot command.
- Keep the project path absolute when working across multiple repos to avoid ambiguity.

## Common log queries

- Recent errors: `flarness logs --level error --since 5m`
- Very recent failures: `flarness logs --since 30s`
- Framework-only failures: `flarness logs --source framework --level error`
- Search a symptom: `flarness logs --grep "overflow" --since 5m`
- Search app output for a feature: `flarness logs --source app --grep "login"`

## Troubleshooting

- Error saying daemon is not running:
  run `flarness start` in the Flutter project or pass `--project`.
- Error saying no `pubspec.yaml`:
  you are not pointing at a Flutter project root.
- Reload appears successful but UI is stale:
  run `flarness restart`.
- `interact tap` or `interact type` succeeded but the UI did not change:
  rerun `flarness semantics`, refocus the target with `interact tap`, then retry the action.
- Text input lands in the wrong field or does not persist:
  explicitly tap the field again, then use `interact type`; verify the field value afterward.
- On macOS, `inspect` may fall back to render tree output instead of a rich widget tree:
  use `semantics` for interaction decisions and treat `inspect` as structural/debugging context.
- On macOS, input focus can drift after rapid interactions:
  slow down the sequence and verify focus-sensitive actions with `semantics` between steps.
- `stop` reports success but you need to be certain the session is clean:
  verify with `flarness status` and, if needed, inspect recent logs before restarting.
- Need machine-readable command schema:
  run `flarness help` or `flarness help <command>`.

## Failure recovery strategy

- If an interaction fails, run `semantics` before retrying so you do not act on stale assumptions.
- If the target is present but the action did not stick, retry with an explicit refocus step: `interact tap` then the intended action.
- If UI state becomes inconsistent after several actions, prefer `restart` over piling on more taps.
- If the daemon socket is unavailable, use `start` instead of retrying subcommands.
- If runtime behavior is unclear, inspect `logs` before changing the UI again.

## Good defaults for an agent using this skill

- Assume JSON output is the source of truth.
- Prefer `screenshot`, `inspect`, `logs`, and `analyze` over guessing what happened in the app.
- When a command fails, surface the structured error payload and choose the next command based on that result.
