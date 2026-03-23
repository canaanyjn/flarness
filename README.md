# Flarness — Flutter AI Harness

Flarness is an **AI-friendly** tool designed to let AI Agents (like coding assistants) drive the complete Flutter development loop. Unlike traditional CLI tools built for humans, Flarness provides **structured JSON responses** for every operation, allowing agents to understand, interact with, and automate Flutter applications with pinpoint precision.

## 🚀 Vision

AI agents are rewriting the development flow. To effectively manage a Flutter app, an agent needs more than just raw shell output; it needs structured state, machine-readable logs, and programmatic control over terminal-based interactions. Flarness provides this bridge.

## ✨ Features

- **JSON-First**: Every command outputs valid JSON, suitable for immediate parsing.
- **Daemon Management**: Run a background flarness daemon to keep the Flutter VM service alive and responsive.
- **Full Control**: Start, stop, restart, and hot-reload Flutter apps programmatically.
- **State Inspection**: Capture UI snapshots, screenshots, and analyze project health in real-time.
- **AI-Friendly Help**: A built-in `help` command that outputs command specifications in JSON.

## 📦 Installation

Required: Go 1.22+

```bash
git clone https://github.com/canaanyjn/flarness.git
cd flarness
make build
# Optional: install to /usr/local/bin
sudo make install
```

## 🛠 Usage

All commands return a JSON object with a `status` field.

### Start the development loop
```bash
flarness start --project /path/to/flutter_project --device chrome
```

### Get current status
```bash
flarness status
```

### Capture a UI Snapshot
```bash
flarness snapshot
```

### Get AI-readable help
```bash
flarness help [command]
```

## 📖 Command list

- `start`: Start the Flarness daemon and launch the Flutter app.
- `stop`: Stop the background daemon and terminate Flutter.
- `reload`: Perform a Hot Reload.
- `restart`: Perform a Hot Restart.
- `screenshot`: Capture an image of the current screen.
- `snapshot`: Get a structured tree of the current UI.
- `logs`: Stream recent records from the application log.
- `status`: Check if the daemon is running and what it's controlling.
- `help`: Get structural information about these commands.

## 🤝 License

[MIT](./LICENSE)
