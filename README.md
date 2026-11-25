# Parla Viva

A fast, GPU-native encrypted desktop messenger and Agent CLI for [Delta Chat / Chatmail](https://delta.chat/), migrated from [Parla](https://github.com/trufae/parla) to Go on top of [Viva](https://github.com/vivaui/viva).

## Highlights

- **Instant First Paint + O(1) Delta Sync**: Local disk snapshot cache (`.parla-viva-cache.json`) renders accounts, chats, and recent messages on Frame 1 before network sync; startup wakes IMAP IDLE/SMTP in parallel at `t=0`, and a non-blocking event pump coalesces rapid bursts while fast-tracking `IncomingMsg`.
- **GoIM-Inspired GPU-Native Desktop UX**: Custom frameless titlebar with live Delta Chat connectivity status, slim left navigation rail (`Chats`, `Contacts`, `Files`, `Pair`, `Theme`, `Settings`), resizable searchable sidebar, right-side docked **Chat Files** drawer, Contact & Group profile cards, workspace-wide **Files & Media** table, categorized Settings with 4 curated light/dark color schemes (`Classic`, `Pine`, `Clay`, `Iris`), docked multi-mode composer with quick emoji bar, multi-image/file drag-and-drop & clipboard paste, and universal in-app previewer (Image lightbox + zoom, Markdown, JSON, CSV/TSV table, line-numbered code, ZIP/XDC archive, vCard).
- **Delta Chat & Parla Compatible**: Works directly with `deltachat-rpc-server` and existing Parla account directories, supporting SecureJoin QR/link pairing, Autocrypt E2EE, Webxdc mini-apps, multi-account switching, and live `get_contacts` / `create_contact` sync.
- **Native OS Notifications**: Asynchronous, per-chat throttled desktop notifications (macOS Notification Center, Windows Toast, Linux `notify-send`) + in-app toasts.
- **Built-in Agent CLI + Local IPC Bridge**: Single binary launches the desktop GUI with no arguments, or runs machine-readable (`--json` / NDJSON) CLI commands with subcommands. While the GUI (or `parla-viva serve`) is running, CLI calls automatically route through an authenticated localhost IPC bridge (`127.0.0.1`) with zero `accounts.lock` conflicts.
- **SafeStore AES-256-GCM Backups**: Encrypted archive creation, SHA-256 manifest verification, and restore from both GUI Settings and CLI.
- **Structured Config & Rotating Logs**: Zero-flag defaults with `parla-viva.json` configuration and size-rotating `slog` logs (`parla-viva config` / `parla-viva logs`).

## Quick Start

Download the latest tri-platform release (`windows-amd64.zip`, `darwin-arm64.tar.gz`, `linux-amd64.tar.gz`) from [Releases](https://github.com/acam-infra/parla-viva/releases):

- **Windows**: Extract `parla-viva.exe` (includes embedded app icon & version metadata) and run it directly, or use `scripts/install-windows.ps1`. Existing `%LOCALAPPDATA%\parla\accounts` and `deltachat-rpc-server.exe` are auto-detected.
- **macOS (Apple Silicon)**: Run `parla-viva` or package into `~/Applications/Parla Viva.app`. Existing Parla accounts (`~/.local/share/parla/accounts` or `~/Library/Application Support/parla/accounts`) are auto-detected.
- **Linux**: Run `./parla-viva` for desktop GUI or `./parla-viva serve` for a headless server IPC daemon.

## Agent CLI Cheat Sheet

Every command supports `--json` for scripts and AI agents (see also `skills/parla-viva-cli/SKILL.md`):

```bash
# Diagnostics (IPC bridge, RPC server, accounts, config & log paths)
parla-viva doctor --json

# Accounts, chats & messages
parla-viva accounts list --json
parla-viva chats list --limit 20 --json
parla-viva messages --chat 12 --limit 20 --json

# Send text, stdin (--text -), replies, or multiple files/images
parla-viva send --chat 12 --text "Status update" --json
parla-viva send --to peer@nine.testrun.org --text "Attached" --file ./chart.png --file ./data.csv --json

# Wait for the next incoming reply (agent request-reply loop)
parla-viva watch --chat 12 --incoming --once --timeout 60s --json

# SecureJoin pairing, file preview & encrypted backups
parla-viva invite --json
parla-viva join --link "https://i.delta.chat/#..." --json
parla-viva preview ./data.csv --json
parla-viva backup create --out ./backup.pvbackup --passphrase "<SECRET>" --json

# Configuration & structured logs
parla-viva config init
parla-viva config show --json
parla-viva config set theme dark
parla-viva config set logLevel debug
parla-viva logs --lines 50
```

See [docs/CONFIGURATION.md](docs/CONFIGURATION.md) for all configuration keys, environment variables, and file locations.

## Building & Testing

```bash
go vet ./...
go test -race -count=1 ./...
go build -trimpath -ldflags="-s -w" -o build/parla-viva ./cmd/parla-viva
```
