# Configuration, Logging & Operations Guide

Parla Viva works out of the box with zero flags, and can be customized through a single JSON config file (`parla-viva.json`), environment variables, or the built-in `parla-viva config` and `parla-viva logs` commands.

## File Locations

All runtime files live cleanly beside your Delta Chat accounts directory (auto-detected per OS):

| Platform | Default `accountsDir` |
| :--- | :--- |
| **Windows** | `%LOCALAPPDATA%\parla\accounts` |
| **macOS** | `~/Library/Application Support/parla/accounts` (if present) or `~/.local/share/parla/accounts` |
| **Linux** | `$XDG_DATA_HOME/parla/accounts` or `~/.local/share/parla/accounts` |

Inside `accountsDir`, Parla Viva manages:

- `parla-viva.json` — User configuration (permissions `0600`)
- `logs/parla-viva.log` — Structured rotating application log (auto-rotates to `.log.1` at `logMaxMB`)
- `.parla-viva-cache.json` — Bounded first-paint snapshot (top 10 chats × 40 messages, `0600`)
- `.parla-viva-ipc.json` — Ephemeral localhost IPC bridge descriptor (`127.0.0.1`, random Bearer token, `0600`)

## Configuration Reference (`parla-viva.json`)

Initialize or inspect your config from any terminal:

```bash
parla-viva config init
parla-viva config show --json
parla-viva config set theme dark
parla-viva config set logLevel debug
parla-viva config set notifications true
```

| Key | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `accountsDir` | string | Auto-detected | Delta Chat accounts directory |
| `rpcServer` | string | Auto-discovered | Explicit path to `deltachat-rpc-server` |
| `theme` | string | `"light"` | UI theme: `"light"`, `"dark"`, or `"system"` |
| `colorScheme` | string | `"classic"` | GoIM-inspired color scheme: `"classic"`, `"pine"`, `"clay"`, or `"iris"` |
| `layout` | string | `"bubbles"` | Message layout: `"bubbles"`, `"workspace"`, or `"compact"` |
| `enterToSend` | bool | `true` | Press Enter to send messages (`false` inserts newline) |
| `notifications` | bool | `true` | Enable native OS desktop notifications on incoming messages |
| `notificationSound` | bool | `true` | Play system notification sound |
| `notifyThrottleMs` | int | `1500` | Per-chat minimum interval (ms) between desktop notifications |
| `logLevel` | string | `"info"` | Structured log level: `"debug"`, `"info"`, `"warn"`, `"error"` |
| `logFile` | string | `<accountsDir>/logs/parla-viva.log` | Log output path |
| `logMaxMB` | int | `10` | Max log file size in MB before automatic rotation |

## Environment Variables

Environment variables override config file values for temporary debugging or headless deployments:

- `DC_ACCOUNTS_PATH` — Override Delta Chat accounts directory
- `PARLA_RPC_SERVER` — Override `deltachat-rpc-server` binary path
- `PARLA_VIVA_CONFIG` — Override `parla-viva.json` file path
- `PARLA_VIVA_LOG_LEVEL` — Override log level (`debug`, `info`, `warn`, `error`)
- `PARLA_VIVA_NOTIFICATIONS` — Override desktop notifications (`true` / `false`)

## Inspecting Logs & Diagnostics

```bash
# Full health check (RPC server, IPC bridge, accounts, config & log paths)
parla-viva doctor --json

# View last 50 structured log lines
parla-viva logs --lines 50

# Filter only warnings or errors
parla-viva logs --lines 100 --filter WARN
parla-viva logs --lines 100 --filter ERROR

# Print log file path
parla-viva logs --path
```
