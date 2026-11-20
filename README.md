# Parla Viva

A Go-first, GPU-native [Delta Chat](https://delta.chat) desktop client migrating [trufae/parla](https://github.com/trufae/parla)'s architecture and UX onto the [vivaui/viva](https://github.com/vivaui/viva) UI framework.

将 [trufae/parla](https://github.com/trufae/parla)（Vala + GTK4/libadwaita）完整迁移至 [vivaui/viva](https://github.com/vivaui/viva)（Go GPU 原生桌面框架：Cocoa/Metal、Win32/Direct3D 11、XCB/Vulkan），通过 stdio JSON-RPC 直连 `deltachat-rpc-server`，彻底摆脱 GTK4 / libadwaita / MSYS2 / AccessKit 跨平台打包复杂度。

## Architecture

- **`internal/dc`**: Stdio JSON-RPC 2.0 transport and typed Delta Chat client (ported from Parla's `rpc_transport.vala` & `rpc_client.vala`), featuring a single writer goroutine, connection-generation invalidation, and `deltachat-rpc-server` auto-discovery.
- **`internal/history`**: Bounded 61-message context window for distant search jumps (`target-30..target+30`), unread-divider initial batching, and bidirectional incremental expansion (ported from Parla's `message_history.vala` / CORE-001).
- **`internal/webxdc`**: Offline `.xdc` ZIP archive server wired to Viva's `app.WebScheme` (`webxdc://app/`), enforcing `Ephemeral: true`, `BridgeMainFrameOnly: true`, path-traversal rejection, `connect-src 'none'` CSP, and a `window.webxdc` bridge over `window.viva.postMessage`.
- **`internal/appui`**: Declarative Viva `ui.C` desktop interface featuring:
  - Resizable split view (`ui.Splitter`) with chatlist search, filters (`All` / `Unread` / `Pinned` / `Groups`), pinned indicators, and unread badges
  - Three Parla conversation layouts (`Bubble` via `ui/message`, `Compact IRC`, and `Workspace`)
  - Pinned messages bar, in-chat search (`Ctrl/Cmd+F`), Quick Switcher (`Ctrl/Cmd+K`), reply previews, emoji reactions, Light/Dark themes, and Settings modal (`Ctrl/Cmd+,`)
  - Interactive offline preview (`--demo`) when `deltachat-rpc-server` is not installed
- **Headless UI Testing**: Driven directly by Viva's `app.NewDriver` with zero display server required (`go test ./...`).

## Quick Start

```sh
# Run all unit, fake-RPC, Webxdc security, and Viva headless UI tests
go test -v ./...

# Launch in interactive offline preview mode (no deltachat-rpc-server needed)
go run ./cmd/parla-viva --demo

# Launch connected to a local deltachat-rpc-server (pip install deltachat-rpc-server)
go run ./cmd/parla-viva
```

## Keyboard Shortcuts

| Action | Linux / Windows | macOS |
| --- | --- | --- |
| Quick chat switcher | `Ctrl+K` | `Command+K` |
| Search in conversation | `Ctrl+F` | `Command+F` |
| New chat | `Ctrl+N` | `Command+N` |
| Settings | `Ctrl+,` | `Command+,` |

## Build & Package

```sh
# Build headless inspection binary
go build -tags vivahead -o build/parla-viva-headless ./cmd/parla-viva

# Build standard release binary
go build -trimpath -ldflags="-s -w" -o build/parla-viva ./cmd/parla-viva
```

## License

Apache License 2.0. Inspired by and migrating architecture from [trufae/parla](https://github.com/trufae/parla), built on [vivaui/viva](https://github.com/vivaui/viva).

## Agent CLI & Local IPC Bridge (`v0.4.0`)

`parla-viva` ships with a built-in Agent CLI designed for both human terminals and AI agents:
- **Zero lock conflicts with GUI**: When the desktop GUI (or `parla-viva serve`) is running, it binds an authenticated localhost IPC bridge (`127.0.0.1`, descriptor stored at `<accountsDir>/.parla-viva-ipc.json` with mode `0600`). CLI commands automatically route through the live bridge so you can query chats, send messages/files, and stream events while the GUI stays open and updates in real time. When no GUI is running (or `--direct` is passed), the CLI automatically launches `deltachat-rpc-server` directly.
- **Machine-readable `--json` / NDJSON output** on every command.

```bash
# Check RPC server, IPC bridge status, and accounts
parla-viva doctor --json

# List accounts & chats
parla-viva accounts list --json
parla-viva chats list --json

# Read recent messages or search
parla-viva messages --chat 12 --limit 20 --json

# Send text (or stdin with --text -), multiple files/images, or reply
parla-viva send --chat 12 --text "Hello from Agent CLI" --json
parla-viva send --to user@nine.testrun.org --text "Report attached" --file ./chart.png --file ./data.csv --json

# Join via SecureJoin link & wait for the next incoming reply (NDJSON)
parla-viva join --link "https://i.delta.chat/#..." --json
parla-viva watch --chat 12 --incoming --once --timeout 60s --json

# Inspect any common file format or manage AES-256-GCM SafeStore backups
parla-viva preview ./data.csv --json
parla-viva backup create --out ./backup.pvbackup --passphrase "..." --json
```
