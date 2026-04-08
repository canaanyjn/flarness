# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

- `make build` — build binary to `bin/flarness` (version injected from git tags via ldflags)
- `make test` — run all tests (`go test ./...`)
- `go test ./internal/parser/...` — run tests for a single package
- `go test -run TestFunctionName ./internal/daemon/` — run a single test
- `make clean` — remove build artifacts

## Architecture

Flarness is a Go CLI tool that lets AI agents drive Flutter apps. It uses a **daemon + CLI architecture** with Unix Domain Socket IPC:

```
AI Agent ──▶ CLI (cobra commands, JSON output) ──▶ Unix Socket ──▶ Daemon ──▶ flutter run --machine
```

### Key packages

- **`cmd/`** — Cobra CLI commands. Each file = one command (start, stop, tap, type, screenshot, etc.). `root.go` has shared JSON output helpers.
- **`internal/daemon/`** — Background daemon that manages the Flutter subprocess and handles IPC requests. The main orchestration layer.
- **`internal/process/`** — Manages `flutter run --machine` subprocess lifecycle (start, stop, stdin forwarding).
- **`internal/parser/`** — Parses `flutter run --machine` stdout (JSON event stream) and stderr (build errors).
- **`internal/collector/`** — Ring-buffer log collection with JSONL file persistence for querying.
- **`internal/inspector/`** — VM Service integration for widget tree inspection. Runs in a separate subprocess to avoid connection conflicts.
- **`internal/interaction/`** — Semantic tree-based UI automation (find widgets by text/label, perform actions). Also subprocess-isolated.
- **`internal/snapshot/`** — Multi-platform screenshot capture (web via CDP, Android via adb, iOS via simctl, macOS via screencapture).
- **`internal/cdp/`** — Chrome DevTools Protocol bridge for web targets.
- **`internal/model/`** — Shared data structures for IPC commands and responses.

### Subprocess isolation pattern

Inspect and interaction commands spawn **separate short-lived processes** that connect to the VM Service independently. This avoids holding a persistent VM Service connection in the daemon, which would conflict with Flutter DevTools. See `internal_inspect.go` and `internal_interact.go` for the subprocess entry points.

### Flutter plugin

`packages/flarness_plugin/` provides debug-mode service extensions (tap, type, swipe, semantics actions) that the tool calls via VM Service. No-ops in release builds.

## Dependencies

Minimal: `gorilla/websocket` (CDP), `spf13/cobra` (CLI), `gopkg.in/yaml.v3` (config). Go 1.22.
