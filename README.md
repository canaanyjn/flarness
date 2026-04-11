# Flarness — Flutter AI Harness

Flarness is an **AI-friendly** tool designed to let AI Agents (like coding assistants) drive the complete Flutter development loop. Unlike traditional CLI tools built for humans, Flarness provides **structured JSON responses** for every operation, allowing agents to understand, interact with, and automate Flutter applications with pinpoint precision.

## 🚀 Vision

AI agents are rewriting the development flow. To effectively manage a Flutter app, an agent needs more than just raw shell output; it needs structured state, machine-readable logs, and programmatic control over terminal-based interactions. Flarness provides this bridge.

## ✨ Features

- **JSON-First**: Every command outputs valid JSON, suitable for immediate parsing.
- **Daemon Management**: Run a background flarness daemon to keep the Flutter VM service alive and responsive.
- **Full Control**: Start, stop, restart, and hot-reload Flutter apps programmatically.
- **State Inspection**: Capture screenshots, inspect UI structure, and analyze project health in real-time.
- **UI Automation**: Drive taps, typing, waits, scrolling, swipes, and long presses through grouped interaction subcommands.
- **AI-Friendly Help**: A built-in `help` command that outputs command specifications in JSON.

## 📦 Installation

Option 1: install a published release on Darwin/Linux

```bash
curl -fsSL https://raw.githubusercontent.com/canaanyjn/flarness/main/release/install.sh | bash
```

Option 2: build from source

Required: Go 1.22+

```bash
git clone https://github.com/canaanyjn/flarness.git
cd flarness
make build
# Optional: install to /usr/local/bin
sudo make install
```

## Skill Install

This repo also exposes a Codex-compatible skill at [`skills/flarness`](/Users/tcn/WorkSpace/Programming/Tools/flarness/skills/flarness/SKILL.md).

With [`vercel-labs/skills`](https://github.com/vercel-labs/skills), install it from the repo directly:

```bash
npx skills add canaanyjn/flarness --skill flarness
```

After that, the installed skill can guide an agent through the Flarness workflow without needing the subdirectory URL.

Note:

- `skills/flarness/` is the canonical install path for external skill installers.

## 🛠 Usage

All commands return a JSON object with a `status` field.

### Start the development loop
```bash
flarness app start --project /path/to/flutter_project --device chrome
```

### Configure a Flutter wrapper command
If your project needs a wrapper instead of calling `flutter` directly, add it to `~/.flarness/config.yaml`:

```yaml
defaults:
  flutter_command:
    - /absolute/path/to/apps/mobile/scripts/dev.sh
  extra_args:
    - --flavor
    - dev
```

Flarness will then execute the configured wrapper and append `run --machine`, the selected device, and any extra args.

### List running sessions
```bash
flarness sessions list
```

### Get current status
```bash
flarness app status --session <session>
```

### Capture a UI Screenshot
```bash
flarness observe screenshot --session <session>
```

On macOS, screenshot support is limited to Flutter-rendered content and
requires the app to initialize
[`flarness_plugin`](/Users/tcn/WorkSpace/Programming/Tools/flarness/packages/flarness_plugin)
in debug mode. It does not capture the desktop, window frame, or native
platform views.

On other non-web platforms, Flarness still tries `flutter screenshot` first and
uses the same plugin-based Flutter-content capture as a fallback when the
native Flutter path fails.

### Inspect the current UI structure
```bash
flarness observe inspect --session <session>
```

### Choose the right view
```text
observe inspect   = structure/debugging view (widget tree or render tree)
observe semantics = automation/interaction view (labels, actions, focus, bounds)
```

### Drive the UI
```bash
flarness interact tap --session <session> --text "Login"
flarness interact type --session <session> --value "hello@example.com"
flarness interact wait --session <session> --text "Success"
```

### Get AI-readable help
```bash
flarness help [command]
```

## 📖 Command list

- `app`: Start, stop, inspect status, reload, and restart the running Flutter app.
- `observe`: Capture screenshots, inspect UI structure, and dump semantics.
- `interact`: Group UI interaction subcommands such as `tap`, `type`, `wait`, `scroll`, `swipe`, and `longpress`.
- `diagnose`: Query logs and run analyzer checks.
- `sessions`: List known daemon sessions.
- `config`: Manage named Flutter project configuration.
- `help`: Get structural information about these commands.

## Flutter App Integration

Interactive commands require the Flutter app to register `ext.flarness.*`
service extensions in debug mode.

Use the bundled Flutter integration package:

- [packages/flarness_plugin](/Users/tcn/WorkSpace/Programming/Tools/flarness/packages/flarness_plugin)
- [docs/flutter-debug-package.md](/Users/tcn/WorkSpace/Programming/Tools/flarness/docs/flutter-debug-package.md)

Recommended dependency for external apps:

```yaml
dependencies:
  flarness_plugin:
    git:
      url: https://github.com/canaanyjn/flarness.git
      path: packages/flarness_plugin
      ref: v0.2.0
```

## 🤝 License

[MIT](./LICENSE)
