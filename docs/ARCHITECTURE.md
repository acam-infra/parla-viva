# Parla → Viva Migration Architecture

## Why Migrate Parla to Viva?

[trufae/parla](https://github.com/trufae/parla) is a clean Vala + GTK4/libadwaita client for [Delta Chat](https://delta.chat), decoupled from the Rust core via `deltachat-rpc-server` over stdio JSON-RPC. However, shipping GTK4 + libadwaita across macOS and Windows requires heavy bundling (custom AccessKit patches for screen readers, MSYS2/UCRT64 toolchains, WebKitGTK AppImage limitations).

[vivaui/viva](https://github.com/vivaui/viva) is a Go-first, GPU-native desktop UI framework (Cocoa/Metal, Win32/D3D11, XCB/Vulkan + WebGL2/Canvas2D) with native accessibility (NSAccessibility, UI Automation, AT-SPI), native IME, zero idle redraws, and single-binary packaging (`viva package`).

## Module Mapping

| Parla (Vala / GTK4) | Parla Viva (Go / Viva) | Notes |
| --- | --- | --- |
| `src/rpc_transport.vala` | `internal/dc/transport.go` | Stdio JSON-RPC 2.0, single writer goroutine, connection generation guard |
| `src/rpc_client.vala` + `rpc_parsers.vala` | `internal/dc/client.go` + `models.go` | Typed accounts, chatlist, messages, reactions, quotes, events |
| `src/message_history.vala` | `internal/history/window.go` | 61-message bounded context window (`target-30..target+30`) & unread divider |
| `src/webxdc*.vala` | `internal/webxdc/host.go` | In-memory `.xdc` ZIP server via `app.WebScheme`, `Ephemeral: true`, `connect-src 'none'` CSP |
| `src/window.vala` + `conversation_view.vala` | `internal/appui/{shell,sidebar,chat_view,contacts_view,files_view,settings_view,modals,theme,state,actions}.go` | Modular GoIM-inspired Viva shell (titlebar, rail, 4 color schemes, right Chat Files drawer, Contacts, Files table, Settings) cleanly decoupled from `internal/dc` |
| `tests/*_ui_test.vala` | `internal/appui/page_test.go` + `screenshot_test.go` | Windowless `app.NewDriver` tests + `viva/headless` offscreen PNG visual QA |

## Verification Pipeline

1. `go vet ./...`
2. `go test -v -count=1 ./...` (fake stdio JSON-RPC server + 61-message window + Webxdc CSP + Viva `app.NewDriver` UI tests)
3. `go test -race -count=1 ./internal/dc ./internal/history ./internal/webxdc`
4. Cross-platform CI builds on Ubuntu, macOS, and Windows via `.github/workflows/ci.yml`
