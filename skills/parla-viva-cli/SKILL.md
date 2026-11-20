---
name: parla-viva-cli
description: Control the Parla Viva encrypted messenger and Delta Chat accounts via its Agent CLI. Use when the user asks to send messages or files over Parla or Delta Chat, check unread chats, read chat history, watch for incoming replies, join SecureJoin invite links, preview attachments, or run headless Chatmail workflows.
---

# Parla Viva CLI

Use `parla-viva` subcommands to interact with Delta Chat / Chatmail accounts from any terminal or AI agent workflow.

## When to Use

- Sending text messages, quotes, or multiple files and images to a Delta Chat contact or group.
- Listing accounts, searching chats, or reading recent messages from an active Parla Viva installation.
- Waiting for an incoming encrypted reply in automated agent request-reply loops.
- Generating or joining SecureJoin invite links (`https://i.delta.chat/#...`).
- Previewing common file formats or creating and verifying AES-256-GCM SafeStore backups.

## Steps

### 1. Check connectivity and IPC bridge mode

Run `doctor` first to confirm whether a desktop GUI or `parla-viva serve` daemon is exposing the local IPC bridge, and to verify configured accounts:

```bash
parla-viva doctor --json
```

When the desktop GUI is open, `parla-viva` automatically routes CLI calls over the authenticated localhost IPC bridge (`127.0.0.1`), avoiding `accounts.lock` conflicts and updating the GUI live. If no GUI is running (or when `--direct` is passed), the CLI spawns `deltachat-rpc-server` directly.

### 2. Inspect accounts and chats

```bash
parla-viva accounts list --json
parla-viva chats list --limit 20 --json
parla-viva messages --chat <CHAT_ID> --limit 20 --json
```

### 3. Send messages and multi-file attachments

Target either an existing `--chat <CHAT_ID>` or an email address `--to <EMAIL>` (which creates or opens the 1:1 chat automatically). Repeat `--file` for multiple images or documents, and pass `--text -` to read the message body from stdin:

```bash
parla-viva send --chat <CHAT_ID> --text "Status update" --json
parla-viva send --to peer@nine.testrun.org --text "Attached" --file ./a.png --file ./report.csv --json
parla-viva send --chat <CHAT_ID> --reply-to <MSG_ID> --text "Confirmed" --json
```

### 4. Wait for a reply (agent request-reply loop)

Use `watch --incoming --once --json` to block until the next incoming message arrives in a chat and print it as a single hydrated JSON object:

```bash
parla-viva watch --chat <CHAT_ID> --incoming --once --timeout 60s --json
```

### 5. Pair via SecureJoin, preview files, or manage backups

```bash
parla-viva invite --json
parla-viva join --link "https://i.delta.chat/#..." --json
parla-viva preview ./data.csv --json
parla-viva backup create --out ./backup.pvbackup --passphrase "<SECRET>" --json
```

## Gotchas

- Always pass `--json` in automated scripts so errors and payloads parse deterministically.
- Do not pass `--direct` while the Parla Viva desktop GUI is running on the same accounts directory, or `deltachat-rpc-server` will contend on `accounts.lock`.
- On headless servers where multiple CLI commands run frequently, start `parla-viva serve` in the background so subsequent commands execute instantly over the local IPC bridge.
