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
- Each Flutter project gets its own Flarness daemon session.
- `start` returns a stable `session` id derived from the project path.
- You usually start once per project, then issue follow-up commands against that session.
- The daemon keeps the Flutter VM service alive and exposes higher-level actions like reload, analyze, inspect, screenshot, logs, and grouped interaction commands.

## Preconditions

- `flarness` should be available on `PATH`.
- Flutter must be installed and usable from the shell.
- For `semantics` and `interact` commands to work, the app should include the
  Flutter-side `flarness_plugin` package in debug builds.
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
  RELEASE_VERSION=v0.1.2 INSTALL_DIR="$HOME/.local/bin" sh
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

1. Start Flarness against the target Flutter project.
2. Capture the returned `session` id from `start`.
3. Use `status --session <session>` until the Flutter process is `running`.
4. Make code changes in the Flutter app.
5. Run `reload --session <session>` for incremental UI changes.
6. Use `screenshot`, `inspect`, and `semantics` with the same `--session`.
7. Use `logs` or `analyze` when reload fails or the UI behaves incorrectly.
8. Use `restart --session <session>` when app state is too stale for a hot reload to recover cleanly.
9. Use `stop --session <session>` when you are done with that project.

## Core commands

- `start`: starts the daemon and launches the Flutter app.
- `sessions list`: lists known daemon sessions and their current state.
- `status --session <session>`: returns whether the daemon session is running and what project/device it controls.
- `reload --session <session>`: sends hot reload.
- `restart --session <session>`: sends hot restart.
- `screenshot --session <session>`: captures the current app screen.
- `inspect --session <session>`: returns the structural debugging view, using widget tree or render tree data.
- `semantics --session <session>`: returns the automation-facing view with labels, actions, focus state, and bounds.
- `logs --session <session>`: queries structured logs.
- `analyze --session <session>`: runs `flutter analyze --no-pub` through Flarness.
- `stop --session <session>`: stops a specific daemon session.
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

List available sessions:

```bash
flarness sessions list
```

Check state:

```bash
flarness status --session <session>
```

Reload after edits:

```bash
flarness reload --session <session>
```

Recover with restart:

```bash
flarness restart --session <session>
```

Inspect widget tree only:

```bash
flarness inspect --session <session> --max-depth 6
```

Dump the automation-facing semantics tree:

```bash
flarness semantics --session <session>
```

Capture only a screenshot:

```bash
flarness screenshot --session <session>
```

Look for recent errors:

```bash
flarness logs --session <session> --level error --since 5m
```

Run analyzer:

```bash
flarness analyze --session <session>
```

Stop the daemon:

```bash
flarness stop --session <session>
```

## Recommended interaction sequence

- Start with `flarness semantics --session <session>` to see what the UI exposes for automation.
- Use `flarness interact tap --session <session>` to focus or select the target element.
- Use `flarness interact type --session <session>` only after focus is confirmed or intentionally set.
- Use `flarness interact wait --session <session>` when the next UI state is expected to appear asynchronously.
- After every write or navigation action, run `flarness semantics --session <session>` again to verify the UI actually changed.
- Use `flarness inspect --session <session>` only when interaction succeeds but the structure or layout still needs explanation.

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
- When working across multiple projects, keep the correct `session` attached to each command.
- Default to `reload`; escalate to `restart` only when necessary.
- Keep commands atomic: call `screenshot` and `inspect` separately when both are needed.
- If you do not know the target session, run `flarness sessions list`.
- If the daemon for a session is not running, call `start` for that project instead of retrying other commands.
- For web devices, screenshot may use CDP internally; otherwise Flarness falls back to Flutter's screenshot command.
- Keep the project path absolute when working across multiple repos to avoid ambiguity.

## Common log queries

- Recent errors: `flarness logs --session <session> --level error --since 5m`
- Very recent failures: `flarness logs --session <session> --since 30s`
- Framework-only failures: `flarness logs --session <session> --source framework --level error`
- Search a symptom: `flarness logs --session <session> --grep "overflow" --since 5m`
- Search app output for a feature: `flarness logs --session <session> --source app --grep "login"`

## Troubleshooting

- Error saying daemon is not running:
  run `flarness sessions list` to confirm the target session, then `flarness start` in the Flutter project if needed.
- `start` fails during startup:
  inspect the daemon log path mentioned in the error; Flarness now waits for daemon IPC and Flutter `running` state before reporting success.
- Error saying no `pubspec.yaml`:
  you are not pointing at a Flutter project root.
- Reload appears successful but UI is stale:
  run `flarness restart --session <session>`.
- `interact tap` or `interact type` succeeded but the UI did not change:
  rerun `flarness semantics --session <session>`, refocus the target with `interact tap --session <session>`, then retry the action.
- Text input lands in the wrong field or does not persist:
  explicitly tap the field again, then use `interact type --session <session>`; verify the field value afterward.
- On macOS, `inspect` may fall back to render tree output instead of a rich widget tree:
  use `semantics` for interaction decisions and treat `inspect` as structural/debugging context.
- On macOS, input focus can drift after rapid interactions:
  slow down the sequence and verify focus-sensitive actions with `semantics` between steps.
- `stop` reports success but you need to be certain the session is clean:
  verify with `flarness status --session <session>` and, if needed, inspect recent logs before restarting.
- Need machine-readable command schema:
  run `flarness help` or `flarness help <command>`.

## Failure recovery strategy

- If an interaction fails, run `semantics` before retrying so you do not act on stale assumptions.
- If the target is present but the action did not stick, retry with an explicit refocus step: `interact tap` then the intended action.
- If UI state becomes inconsistent after several actions, prefer `restart` over piling on more taps.
- If the daemon socket is unavailable, confirm the target session with `sessions list`, then use `start` instead of retrying subcommands blindly.
- If runtime behavior is unclear, inspect `logs` before changing the UI again.

## Good defaults for an agent using this skill

- Assume JSON output is the source of truth.
- Prefer `screenshot`, `inspect`, `logs`, and `analyze` over guessing what happened in the app.
- When a command fails, surface the structured error payload and choose the next command based on that result.
