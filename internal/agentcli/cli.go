package agentcli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/acam-infra/parla-viva/internal/applog"
	"github.com/acam-infra/parla-viva/internal/backup"
	"github.com/acam-infra/parla-viva/internal/config"
	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/preview"
)

// Version is the current Parla Viva CLI & app version.
const Version = "0.8.0"

// knownCommands lists top-level subcommands handled by the Agent CLI.
var knownCommands = map[string]bool{
	"cli":      true,
	"help":     true,
	"-h":       true,
	"--help":   true,
	"version":  true,
	"doctor":   true,
	"accounts": true,
	"chats":    true,
	"messages": true,
	"msgs":     true,
	"send":     true,
	"react":    true,
	"edit":     true,
	"delete":   true,
	"invite":   true,
	"join":     true,
	"watch":    true,
	"preview":  true,
	"backup":   true,
	"config":   true,
	"logs":     true,
	"serve":    true,
}

// ShouldHandle reports whether os.Args[1:] invokes an Agent CLI subcommand
// rather than launching the Viva desktop GUI.
func ShouldHandle(args []string) bool {
	if len(args) == 0 {
		return false
	}
	// Scan past leading global flags like --json, --accounts dir, --rpc path, --direct, --account 1.
	i := 0
	for i < len(args) {
		a := args[i]
		if knownCommands[a] {
			return true
		}
		if a == "--json" || a == "--direct" {
			i++
			continue
		}
		if a == "--accounts" || a == "-accounts" || a == "--rpc" || a == "-rpc" || a == "--account" || a == "--timeout" {
			i += 2
			continue
		}
		if strings.HasPrefix(a, "--accounts=") || strings.HasPrefix(a, "-accounts=") ||
			strings.HasPrefix(a, "--rpc=") || strings.HasPrefix(a, "-rpc=") ||
			strings.HasPrefix(a, "--account=") || strings.HasPrefix(a, "--timeout=") {
			i++
			continue
		}
		// Any other flag (e.g. --demo) belongs to GUI launch unless a known command follows.
		if strings.HasPrefix(a, "-") {
			i++
			continue
		}
		return false
	}
	return false
}

// Options configures CLI execution (injectable for tests).
type Options struct {
	Stdin       io.Reader
	Stdout      io.Writer
	Stderr      io.Writer
	Client      *dc.Client    // optional pre-connected client for tests
	IPC         *dc.IPCClient // optional pre-connected IPC client for tests
	AccountsDir string
	RPCPath     string
}

type globalOpts struct {
	jsonOut     bool
	direct      bool
	accountID   int
	accountsDir string
	rpcPath     string
	timeout     time.Duration
}

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// Run executes the Agent CLI with args (typically os.Args[1:]) and returns an exit code.
func Run(ctx context.Context, args []string, opts Options) int {
	if opts.Stdin == nil {
		opts.Stdin = os.Stdin
	}
	if opts.Stdout == nil {
		opts.Stdout = os.Stdout
	}
	if opts.Stderr == nil {
		opts.Stderr = os.Stderr
	}

	g := globalOpts{
		accountsDir: opts.AccountsDir,
		rpcPath:     opts.RPCPath,
		timeout:     30 * time.Second,
	}
	if g.accountsDir == "" {
		g.accountsDir = dc.DefaultAccountsDir()
	}

	// Parse global flags that appear before the subcommand.
	cmdIdx := 0
	for cmdIdx < len(args) {
		a := args[cmdIdx]
		if knownCommands[a] {
			break
		}
		switch {
		case a == "--json":
			g.jsonOut = true
			cmdIdx++
		case a == "--direct":
			g.direct = true
			cmdIdx++
		case a == "--accounts" || a == "-accounts":
			if cmdIdx+1 < len(args) {
				g.accountsDir = args[cmdIdx+1]
				cmdIdx += 2
			} else {
				cmdIdx++
			}
		case strings.HasPrefix(a, "--accounts="):
			g.accountsDir = strings.TrimPrefix(a, "--accounts=")
			cmdIdx++
		case a == "--rpc" || a == "-rpc":
			if cmdIdx+1 < len(args) {
				g.rpcPath = args[cmdIdx+1]
				cmdIdx += 2
			} else {
				cmdIdx++
			}
		case strings.HasPrefix(a, "--rpc="):
			g.rpcPath = strings.TrimPrefix(a, "--rpc=")
			cmdIdx++
		case a == "--account":
			if cmdIdx+1 < len(args) {
				g.accountID, _ = strconv.Atoi(args[cmdIdx+1])
				cmdIdx += 2
			} else {
				cmdIdx++
			}
		case strings.HasPrefix(a, "--account="):
			g.accountID, _ = strconv.Atoi(strings.TrimPrefix(a, "--account="))
			cmdIdx++
		case a == "--timeout":
			if cmdIdx+1 < len(args) {
				if d, err := time.ParseDuration(args[cmdIdx+1]); err == nil {
					g.timeout = d
				}
				cmdIdx += 2
			} else {
				cmdIdx++
			}
		case strings.HasPrefix(a, "--timeout="):
			if d, err := time.ParseDuration(strings.TrimPrefix(a, "--timeout=")); err == nil {
				g.timeout = d
			}
			cmdIdx++
		default:
			cmdIdx++
		}
	}

	if cmdIdx >= len(args) {
		printUsage(opts.Stdout)
		return 0
	}
	cmd := args[cmdIdx]
	rest := args[cmdIdx+1:]
	if cmd == "cli" {
		if len(rest) == 0 {
			printUsage(opts.Stdout)
			return 0
		}
		cmd = rest[0]
		rest = rest[1:]
	}

	// Extract trailing global flags (--json, --direct, --accounts, --rpc, --account, --timeout)
	// so users and agents can place them anywhere on the command line.
	rest = extractGlobalFlags(rest, &g)

	var err error
	switch cmd {
	case "help", "-h", "--help":
		printUsage(opts.Stdout)
		return 0
	case "version":
		err = writeOut(opts.Stdout, g.jsonOut, map[string]any{"version": Version}, fmt.Sprintf("parla-viva %s\n", Version))
	case "preview":
		err = cmdPreview(rest, g, opts)
	case "backup":
		err = cmdBackup(rest, g, opts)
	case "config":
		err = cmdConfig(rest, g, opts)
	case "logs":
		err = cmdLogs(rest, g, opts)
	case "doctor":
		err = cmdDoctor(ctx, g, opts)
	case "serve":
		err = cmdServe(ctx, g, opts)
	case "accounts":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdAccounts(cctx, client, rest, g, opts)
		})
	case "chats":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdChats(cctx, client, rest, g, opts)
		})
	case "messages", "msgs":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdMessages(cctx, client, rest, g, opts)
		})
	case "send":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdSend(cctx, client, rest, g, opts)
		})
	case "react":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdReact(cctx, client, rest, g, opts)
		})
	case "edit":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdEdit(cctx, client, rest, g, opts)
		})
	case "delete":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdDelete(cctx, client, rest, g, opts)
		})
	case "invite":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdInvite(cctx, client, rest, g, opts)
		})
	case "join":
		err = withClient(ctx, g, opts, func(cctx context.Context, client *dc.Client, _ *dc.IPCClient) error {
			return cmdJoin(cctx, client, rest, g, opts)
		})
	case "watch":
		err = cmdWatch(ctx, rest, g, opts)
	default:
		err = fmt.Errorf("unknown command %q (run 'parla-viva help')", cmd)
	}

	if err != nil {
		if g.jsonOut {
			_ = json.NewEncoder(opts.Stdout).Encode(map[string]any{"ok": false, "error": err.Error()})
		} else {
			fmt.Fprintf(opts.Stderr, "error: %v\n", err)
		}
		return 1
	}
	return 0
}

func extractGlobalFlags(args []string, g *globalOpts) []string {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--json":
			g.jsonOut = true
		case a == "--direct":
			g.direct = true
		case (a == "--accounts" || a == "-accounts") && i+1 < len(args):
			g.accountsDir = args[i+1]
			i++
		case strings.HasPrefix(a, "--accounts="):
			g.accountsDir = strings.TrimPrefix(a, "--accounts=")
		case (a == "--rpc" || a == "-rpc") && i+1 < len(args):
			g.rpcPath = args[i+1]
			i++
		case strings.HasPrefix(a, "--rpc="):
			g.rpcPath = strings.TrimPrefix(a, "--rpc=")
		case a == "--account" && i+1 < len(args):
			g.accountID, _ = strconv.Atoi(args[i+1])
			i++
		case strings.HasPrefix(a, "--account="):
			g.accountID, _ = strconv.Atoi(strings.TrimPrefix(a, "--account="))
		case a == "--timeout" && i+1 < len(args):
			if d, err := time.ParseDuration(args[i+1]); err == nil {
				g.timeout = d
			}
			i++
		case strings.HasPrefix(a, "--timeout="):
			if d, err := time.ParseDuration(strings.TrimPrefix(a, "--timeout=")); err == nil {
				g.timeout = d
			}
		default:
			out = append(out, a)
		}
	}
	return out
}

func connectClient(ctx context.Context, g globalOpts, opts Options) (*dc.Client, *dc.IPCClient, func(), string, error) {
	if opts.Client != nil {
		return opts.Client, opts.IPC, func() {}, "injected", nil
	}
	if !g.direct {
		if ipcClient, err := dc.DiscoverIPC(ctx, g.accountsDir); err == nil && ipcClient != nil {
			return dc.NewClientWithCaller(ipcClient), ipcClient, func() {}, "ipc:" + ipcClient.State().Addr, nil
		}
	}
	bin := dc.DiscoverRPCServer(g.rpcPath)
	if bin == "" {
		return nil, nil, func() {}, "", errors.New("deltachat-rpc-server not found (pass --rpc or add it to PATH)")
	}
	tr := dc.NewTransport()
	if err := tr.StartProcess(bin, g.accountsDir); err != nil {
		return nil, nil, func() {}, "", err
	}
	client := dc.NewClient(tr)
	_ = client.StartIOForAllAccounts(ctx)
	return client, nil, func() { _ = tr.Close() }, "direct:" + bin, nil
}

func withClient(ctx context.Context, g globalOpts, opts Options, fn func(context.Context, *dc.Client, *dc.IPCClient) error) error {
	cctx := ctx
	var cancel context.CancelFunc
	if g.timeout > 0 {
		cctx, cancel = context.WithTimeout(ctx, g.timeout)
		defer cancel()
	}
	client, ipcClient, cleanup, _, err := connectClient(cctx, g, opts)
	if err != nil {
		return err
	}
	defer cleanup()
	return fn(cctx, client, ipcClient)
}

func resolveAccountID(ctx context.Context, client *dc.Client, requested int) (int, error) {
	if requested > 0 {
		return requested, nil
	}
	if sel, err := client.GetSelectedAccountID(ctx); err == nil && sel > 0 {
		return sel, nil
	}
	accounts, err := client.GetAllAccounts(ctx)
	if err != nil {
		return 0, err
	}
	for _, a := range accounts {
		if a.Kind == "Configured" {
			return a.ID, nil
		}
	}
	if len(accounts) > 0 {
		return accounts[0].ID, nil
	}
	return 0, errors.New("no Delta Chat accounts found (run 'parla-viva accounts create --name ...')")
}

func hydrateAccounts(ctx context.Context, client *dc.Client) ([]dc.Account, error) {
	accounts, err := client.GetAllAccounts(ctx)
	if err != nil {
		return nil, err
	}
	for i := range accounts {
		if accounts[i].DisplayName == "" {
			if dn, err := client.GetConfig(ctx, accounts[i].ID, "displayname"); err == nil && dn != "" {
				accounts[i].DisplayName = dn
			}
		}
		if accounts[i].Addr == "" {
			if addr, err := client.GetConfig(ctx, accounts[i].ID, "addr"); err == nil && addr != "" {
				accounts[i].Addr = addr
			}
		}
	}
	return accounts, nil
}

func writeOut(w io.Writer, asJSON bool, payload any, text string) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}
	_, err := io.WriteString(w, text)
	return err
}

func cmdDoctor(ctx context.Context, g globalOpts, opts Options) error {
	rpcBin := dc.DiscoverRPCServer(g.rpcPath)
	ipcClient, _ := dc.DiscoverIPC(ctx, g.accountsDir)
	mode := "direct"
	ipcAddr := ""
	ipcPID := 0
	if ipcClient != nil && !g.direct {
		mode = "ipc"
		ipcAddr = ipcClient.State().Addr
		ipcPID = ipcClient.State().PID
	}

	appCfg, cfgPath, _ := config.Load(g.accountsDir)
	info := map[string]any{
		"ok":          true,
		"version":     Version,
		"accountsDir": g.accountsDir,
		"configPath":  cfgPath,
		"logFile":     appCfg.LogFile,
		"logLevel":    appCfg.LogLevel,
		"rpcServer":   rpcBin,
		"mode":        mode,
		"ipcAddr":     ipcAddr,
		"ipcPid":      ipcPID,
	}

	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	client, _, cleanup, connDesc, err := connectClient(cctx, g, opts)
	if err != nil {
		info["ok"] = false
		info["error"] = err.Error()
	} else {
		defer cleanup()
		info["connection"] = connDesc
		if accts, aerr := hydrateAccounts(cctx, client); aerr == nil {
			info["accounts"] = accts
			if len(accts) > 0 {
				if conn, cerr := client.GetConnectivity(cctx, accts[0].ID); cerr == nil {
					info["connectivity"] = conn
				}
			}
		} else {
			info["accountsError"] = aerr.Error()
		}
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "Parla Viva Agent CLI v%s\n", Version)
	fmt.Fprintf(&sb, "  Accounts Dir : %s\n", g.accountsDir)
	fmt.Fprintf(&sb, "  RPC Server   : %s\n", rpcBin)
	fmt.Fprintf(&sb, "  Mode         : %s", mode)
	if ipcAddr != "" {
		fmt.Fprintf(&sb, " (GUI/daemon PID %d @ %s)", ipcPID, ipcAddr)
	}
	sb.WriteString("\n")
	if accts, ok := info["accounts"].([]dc.Account); ok {
		fmt.Fprintf(&sb, "  Accounts     : %d\n", len(accts))
		for _, a := range accts {
			fmt.Fprintf(&sb, "    - #%d [%s] %s <%s>\n", a.ID, a.Kind, a.DisplayName, a.Addr)
		}
	}
	if e, ok := info["error"].(string); ok && e != "" {
		fmt.Fprintf(&sb, "  Status       : ERROR (%s)\n", e)
	}
	return writeOut(opts.Stdout, g.jsonOut, info, sb.String())
}

func cmdAccounts(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	sub := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	switch sub {
	case "list":
		accts, err := hydrateAccounts(ctx, client)
		if err != nil {
			return err
		}
		selID, _ := client.GetSelectedAccountID(ctx)
		var sb strings.Builder
		for _, a := range accts {
			mark := " "
			if a.ID == selID {
				mark = "*"
			}
			fmt.Fprintf(&sb, "%s #%d\t%s\t%s\t<%s>\n", mark, a.ID, a.Kind, a.DisplayName, a.Addr)
		}
		if len(accts) == 0 {
			sb.WriteString("(no accounts)\n")
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{
			"selectedAccountId": selID,
			"accounts":          accts,
		}, sb.String())

	case "create":
		fs := flag.NewFlagSet("accounts create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		name := fs.String("name", "Parla Agent", "display name")
		relay := fs.String("relay", dc.DefaultChatmailRelay, "Chatmail relay URI (dcaccount:...)")
		if err := fs.Parse(args); err != nil {
			return err
		}
		id, qr, err := client.CreateChatmailAccount(ctx, *name, *relay)
		if err != nil {
			return err
		}
		addr, _ := client.GetConfig(ctx, id, "addr")
		payload := map[string]any{
			"ok":          true,
			"accountId":   id,
			"displayName": *name,
			"addr":        addr,
			"inviteLink":  qr,
		}
		return writeOut(opts.Stdout, g.jsonOut, payload, fmt.Sprintf("Created account #%d (%s <%s>)\nInvite: %s\n", id, *name, addr, qr))

	case "select":
		fs := flag.NewFlagSet("accounts select", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		id := fs.Int("id", 0, "account ID")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *id <= 0 && fs.NArg() > 0 {
			*id, _ = strconv.Atoi(fs.Arg(0))
		}
		if *id <= 0 {
			return errors.New("accounts select requires --id <accountId>")
		}
		if err := client.SelectAccount(ctx, *id); err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "selectedAccountId": *id}, fmt.Sprintf("Selected account #%d\n", *id))

	default:
		return fmt.Errorf("unknown accounts subcommand %q (use list, create, select)", sub)
	}
}

func cmdChats(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	sub := "list"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}

	switch sub {
	case "list":
		fs := flag.NewFlagSet("chats list", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		query := fs.String("query", "", "filter chats by query")
		limit := fs.Int("limit", 50, "maximum chats to return")
		if err := fs.Parse(args); err != nil {
			return err
		}
		ids, err := client.GetChatlistEntries(ctx, acctID, nil, *query)
		if err != nil {
			return err
		}
		if *limit > 0 && len(ids) > *limit {
			ids = ids[:*limit]
		}
		chats, err := client.GetChatlistItems(ctx, acctID, ids)
		if err != nil {
			return err
		}
		var sb strings.Builder
		for _, ch := range chats {
			flags := []string{}
			if ch.IsPinned {
				flags = append(flags, "pinned")
			}
			if ch.IsMuted {
				flags = append(flags, "muted")
			}
			if ch.IsContactRequest {
				flags = append(flags, "request")
			}
			flagStr := ""
			if len(flags) > 0 {
				flagStr = " [" + strings.Join(flags, ",") + "]"
			}
			unread := ""
			if ch.UnreadCount > 0 {
				unread = fmt.Sprintf(" (%d unread)", ch.UnreadCount)
			}
			fmt.Fprintf(&sb, "#%d\t%s%s%s\t%s\n", ch.ID, ch.Name, flagStr, unread, strings.ReplaceAll(ch.LastMessage, "\n", " "))
		}
		if len(chats) == 0 {
			sb.WriteString("(no chats)\n")
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{
			"accountId": acctID,
			"count":     len(chats),
			"chats":     chats,
		}, sb.String())

	case "info":
		fs := flag.NewFlagSet("chats info", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		chatID := fs.Int("chat", 0, "chat ID")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *chatID <= 0 && fs.NArg() > 0 {
			*chatID, _ = strconv.Atoi(fs.Arg(0))
		}
		if *chatID <= 0 {
			return errors.New("chats info requires --chat <id>")
		}
		info, err := client.GetBasicChatInfo(ctx, acctID, *chatID)
		if err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, info, fmt.Sprintf("Chat #%d: %s (%s, encrypted=%v, pinned=%v, muted=%v)\n",
			info.ID, info.Name, info.ChatType, info.IsEncrypted, info.Pinned, info.IsMuted))

	case "create-group":
		fs := flag.NewFlagSet("chats create-group", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		name := fs.String("name", "", "group name")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if strings.TrimSpace(*name) == "" {
			return errors.New("chats create-group requires --name <groupName>")
		}
		chatID, err := client.CreateGroupChat(ctx, acctID, *name, false)
		if err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "chatId": chatID, "name": *name}, fmt.Sprintf("Created group #%d (%s)\n", chatID, *name))

	case "accept", "pin", "unpin", "archive", "mute", "unmute":
		fs := flag.NewFlagSet("chats "+sub, flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		chatID := fs.Int("chat", 0, "chat ID")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if *chatID <= 0 && fs.NArg() > 0 {
			*chatID, _ = strconv.Atoi(fs.Arg(0))
		}
		if *chatID <= 0 {
			return fmt.Errorf("chats %s requires --chat <id>", sub)
		}
		switch sub {
		case "accept":
			err = client.AcceptChat(ctx, acctID, *chatID)
		case "pin":
			err = client.SetChatVisibility(ctx, acctID, *chatID, "Pinned")
		case "unpin":
			err = client.SetChatVisibility(ctx, acctID, *chatID, "Normal")
		case "archive":
			err = client.SetChatVisibility(ctx, acctID, *chatID, "Archived")
		case "mute":
			err = client.SetChatMuteDuration(ctx, acctID, *chatID, true)
		case "unmute":
			err = client.SetChatMuteDuration(ctx, acctID, *chatID, false)
		}
		if err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "action": sub, "chatId": *chatID}, fmt.Sprintf("Chat #%d: %s ok\n", *chatID, sub))

	default:
		return fmt.Errorf("unknown chats subcommand %q", sub)
	}
}

func cmdMessages(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("messages", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	chatID := fs.Int("chat", 0, "chat ID")
	msgID := fs.Int("id", 0, "single message ID")
	query := fs.String("query", "", "search messages by text")
	limit := fs.Int("limit", 30, "maximum recent messages")
	markSeen := fs.Bool("mark-seen", false, "mark returned messages as seen")
	if err := fs.Parse(args); err != nil {
		return err
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}

	var ids []int
	switch {
	case *msgID > 0:
		ids = []int{*msgID}
	case *query != "":
		ids, err = client.SearchMessages(ctx, acctID, *chatID, *query)
		if err != nil {
			return err
		}
	case *chatID > 0:
		ids, err = client.GetMessageIDs(ctx, acctID, *chatID)
		if err != nil {
			return err
		}
	default:
		return errors.New("messages requires --chat <id>, --id <msgId>, or --query <text>")
	}

	if *limit > 0 && len(ids) > *limit {
		ids = ids[len(ids)-*limit:]
	}
	msgs, err := client.GetMessages(ctx, acctID, ids)
	if err != nil {
		return err
	}
	if *markSeen && len(ids) > 0 {
		_ = client.MarkSeenMsgs(ctx, acctID, ids)
	}

	var sb strings.Builder
	for _, m := range msgs {
		dir := "<-"
		if m.IsOutgoing {
			dir = "->"
		}
		fileNote := ""
		if m.FileName != "" {
			fileNote = fmt.Sprintf(" [file: %s]", m.FileName)
		} else if m.FilePath != "" {
			fileNote = fmt.Sprintf(" [file: %s]", filepath.Base(m.FilePath))
		}
		fmt.Fprintf(&sb, "#%d %s [%s] %s: %s%s\n", m.ID, dir, time.Unix(m.Timestamp, 0).Format("15:04"), m.DisplaySender(), strings.ReplaceAll(m.Text, "\n", "\\n"), fileNote)
	}
	if len(msgs) == 0 {
		sb.WriteString("(no messages)\n")
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{
		"accountId": acctID,
		"chatId":    *chatID,
		"count":     len(msgs),
		"messages":  msgs,
	}, sb.String())
}

func cmdSend(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	chatID := fs.Int("chat", 0, "target chat ID")
	toEmail := fs.String("to", "", "recipient email address (creates/opens 1:1 chat)")
	text := fs.String("text", "", "message text (use '-' to read from stdin)")
	replyTo := fs.Int("reply-to", 0, "message ID to quote/reply to")
	var files stringList
	fs.Var(&files, "file", "file/image attachment path (repeatable)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *text == "" && fs.NArg() > 0 {
		*text = strings.Join(fs.Args(), " ")
	}
	if *text == "-" {
		raw, err := io.ReadAll(opts.Stdin)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		*text = strings.TrimRight(string(raw), "\r\n")
	}
	if strings.TrimSpace(*text) == "" && len(files) == 0 {
		return errors.New("send requires --text <message> and/or --file <path>")
	}

	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}
	targetChat := *chatID
	if targetChat <= 0 {
		if strings.TrimSpace(*toEmail) == "" {
			return errors.New("send requires --chat <id> or --to <email>")
		}
		contactID, err := client.CreateContact(ctx, acctID, strings.TrimSpace(*toEmail), "")
		if err != nil {
			return err
		}
		targetChat, err = client.CreateChatByContactID(ctx, acctID, contactID)
		if err != nil {
			return err
		}
	}

	var sentIDs []int
	if len(files) == 0 {
		msgID, err := client.SendTextMessage(ctx, acctID, targetChat, *text, *replyTo)
		if err != nil {
			return err
		}
		sentIDs = append(sentIDs, msgID)
	} else {
		for idx, fp := range files {
			absPath, err := filepath.Abs(fp)
			if err != nil {
				return err
			}
			if _, err := os.Stat(absPath); err != nil {
				return fmt.Errorf("attachment %q: %w", fp, err)
			}
			caption := ""
			quote := 0
			if idx == 0 {
				caption = *text
				quote = *replyTo
			}
			msgID, err := client.SendFileMessageWithQuote(ctx, acctID, targetChat, absPath, caption, quote)
			if err != nil {
				return err
			}
			sentIDs = append(sentIDs, msgID)
		}
	}

	payload := map[string]any{
		"ok":        true,
		"accountId": acctID,
		"chatId":    targetChat,
		"msgIds":    sentIDs,
	}
	return writeOut(opts.Stdout, g.jsonOut, payload, fmt.Sprintf("Sent %d message(s) to chat #%d: %v\n", len(sentIDs), targetChat, sentIDs))
}

func cmdReact(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("react", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	msgID := fs.Int("msg", 0, "message ID")
	emoji := fs.String("emoji", "", "emoji reaction (empty to clear)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *msgID <= 0 {
		return errors.New("react requires --msg <id>")
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}
	var emojis []string
	if strings.TrimSpace(*emoji) != "" {
		emojis = []string{strings.TrimSpace(*emoji)}
	}
	if err := client.SendReaction(ctx, acctID, *msgID, emojis...); err != nil {
		return err
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "msgId": *msgID, "emoji": *emoji}, fmt.Sprintf("Reacted to #%d with %q\n", *msgID, *emoji))
}

func cmdEdit(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("edit", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	msgID := fs.Int("msg", 0, "message ID")
	text := fs.String("text", "", "new message text")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *msgID <= 0 || strings.TrimSpace(*text) == "" {
		return errors.New("edit requires --msg <id> and --text <newText>")
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}
	if err := client.SendEditRequest(ctx, acctID, *msgID, *text); err != nil {
		return err
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "msgId": *msgID}, fmt.Sprintf("Edited message #%d\n", *msgID))
}

func cmdDelete(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("delete", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var rawIDs stringList
	fs.Var(&rawIDs, "msg", "message ID (repeatable or comma-separated)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	var ids []int
	for _, s := range append(rawIDs, fs.Args()...) {
		for _, part := range strings.Split(s, ",") {
			if id, err := strconv.Atoi(strings.TrimSpace(part)); err == nil && id > 0 {
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return errors.New("delete requires --msg <id>")
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}
	if err := client.DeleteMessages(ctx, acctID, ids); err != nil {
		return err
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "deleted": ids}, fmt.Sprintf("Deleted message(s): %v\n", ids))
}

func cmdInvite(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("invite", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	chatID := fs.Int("chat", 0, "optional group chat ID (0 = account invite)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}
	qr, err := client.GetChatSecurejoinQRCode(ctx, acctID, *chatID)
	if err != nil {
		return err
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{
		"ok":         true,
		"accountId":  acctID,
		"chatId":     *chatID,
		"inviteLink": qr,
	}, qr+"\n")
}

func cmdJoin(ctx context.Context, client *dc.Client, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("join", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	link := fs.String("link", "", "SecureJoin invite link (https://i.delta.chat/#...)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *link == "" && fs.NArg() > 0 {
		*link = fs.Arg(0)
	}
	if strings.TrimSpace(*link) == "" {
		return errors.New("join requires --link <inviteUrl>")
	}
	acctID, err := resolveAccountID(ctx, client, g.accountID)
	if err != nil {
		return err
	}
	chatID, err := client.SecureJoin(ctx, acctID, strings.TrimSpace(*link))
	if err != nil {
		return err
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{
		"ok":        true,
		"accountId": acctID,
		"chatId":    chatID,
	}, fmt.Sprintf("Joined encrypted chat #%d\n", chatID))
}

func cmdWatch(ctx context.Context, args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("watch", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	chatID := fs.Int("chat", 0, "filter events by chat ID")
	incomingOnly := fs.Bool("incoming", false, "only emit IncomingMsg events (hydrated)")
	once := fs.Bool("once", false, "exit after first matching event")
	timeout := fs.Duration("timeout", 0, "optional watch timeout (e.g. 60s; 0 = forever)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	wctx := ctx
	var cancel context.CancelFunc
	if *timeout > 0 {
		wctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	client, ipcClient, cleanup, _, err := connectClient(wctx, g, opts)
	if err != nil {
		return err
	}
	defer cleanup()

	emit := func(ev dc.Event) bool {
		if *chatID > 0 && ev.Event.ChatID != *chatID {
			return true
		}
		if *incomingOnly && ev.Event.Kind != "IncomingMsg" {
			return true
		}
		var hydrated *dc.Message
		if ev.Event.Kind == "IncomingMsg" && ev.Event.MsgID > 0 {
			if msgs, merr := client.GetMessages(wctx, ev.ContextID, []int{ev.Event.MsgID}); merr == nil && len(msgs) > 0 {
				hydrated = &msgs[0]
			}
		}
		if g.jsonOut {
			row := map[string]any{
				"accountId": ev.ContextID,
				"kind":      ev.Event.Kind,
				"chatId":    ev.Event.ChatID,
				"msgId":     ev.Event.MsgID,
			}
			if hydrated != nil {
				row["message"] = hydrated
			}
			_ = json.NewEncoder(opts.Stdout).Encode(row)
		} else if hydrated != nil {
			fmt.Fprintf(opts.Stdout, "[%s] chat=#%d msg=#%d %s: %s\n", ev.Event.Kind, ev.Event.ChatID, ev.Event.MsgID, hydrated.SenderName, hydrated.Text)
		} else {
			fmt.Fprintf(opts.Stdout, "[%s] acct=#%d chat=#%d msg=#%d\n", ev.Event.Kind, ev.ContextID, ev.Event.ChatID, ev.Event.MsgID)
		}
		return !*once
	}

	if ipcClient != nil {
		err := ipcClient.StreamEvents(wctx, emit)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return err
		}
		return err
	}

	for {
		ev, err := client.GetNextEvent(wctx)
		if err != nil {
			return err
		}
		if !emit(ev) {
			return nil
		}
	}
}

func cmdPreview(args []string, g globalOpts, opts Options) error {
	if len(args) == 0 {
		return errors.New("preview requires <filePath>")
	}
	doc, err := preview.LoadFile(args[0])
	if err != nil {
		return err
	}
	if g.jsonOut {
		return writeOut(opts.Stdout, true, doc, "")
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s (%s, %d B)\n", doc.Name, doc.Kind, doc.SizeBytes)
	if doc.Text != "" {
		sb.WriteString("\n" + doc.Text + "\n")
	}
	if len(doc.Headers) > 0 {
		sb.WriteString("\n" + strings.Join(doc.Headers, " | ") + "\n")
		for _, row := range doc.Rows {
			sb.WriteString(strings.Join(row, " | ") + "\n")
		}
	}
	if len(doc.ArchiveItems) > 0 {
		for _, it := range doc.ArchiveItems {
			fmt.Fprintf(&sb, "  %s (%d B)\n", it.Name, it.Size)
		}
	}
	if doc.VCard != nil {
		fmt.Fprintf(&sb, "  Name: %s\n  Email: %s\n  Phone: %s\n  Org: %s\n", doc.VCard.FullName, doc.VCard.Email, doc.VCard.Phone, doc.VCard.Org)
	}
	return writeOut(opts.Stdout, false, nil, sb.String())
}

func cmdBackup(args []string, g globalOpts, opts Options) error {
	if len(args) == 0 {
		return errors.New("backup requires subcommand: create | verify | restore")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "create":
		fs := flag.NewFlagSet("backup create", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		out := fs.String("out", "", "output .pvbackup path")
		pass := fs.String("passphrase", "", "AES-256-GCM passphrase")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *out == "" || *pass == "" {
			return errors.New("backup create requires --out <file> --passphrase <secret>")
		}
		manifest, err := backup.Create(g.accountsDir, *out, *pass)
		if err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "out": *out, "manifest": manifest},
			fmt.Sprintf("Encrypted backup created: %s (%d files)\n", *out, manifest.FileCount))

	case "verify":
		fs := flag.NewFlagSet("backup verify", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		in := fs.String("in", "", "input .pvbackup path")
		pass := fs.String("passphrase", "", "AES-256-GCM passphrase")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *in == "" || *pass == "" {
			return errors.New("backup verify requires --in <file> --passphrase <secret>")
		}
		manifest, err := backup.Verify(*in, *pass)
		if err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "manifest": manifest},
			fmt.Sprintf("Backup verified: %d files (created %s)\n", manifest.FileCount, manifest.CreatedAt))

	case "restore":
		fs := flag.NewFlagSet("backup restore", flag.ContinueOnError)
		fs.SetOutput(io.Discard)
		in := fs.String("in", "", "input .pvbackup path")
		outDir := fs.String("out", g.accountsDir, "destination directory")
		pass := fs.String("passphrase", "", "AES-256-GCM passphrase")
		if err := fs.Parse(rest); err != nil {
			return err
		}
		if *in == "" || *pass == "" {
			return errors.New("backup restore requires --in <file> --passphrase <secret>")
		}
		manifest, err := backup.Restore(*in, *outDir, *pass)
		if err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "out": *outDir, "manifest": manifest},
			fmt.Sprintf("Restored %d files to %s\n", manifest.FileCount, *outDir))

	default:
		return fmt.Errorf("unknown backup subcommand %q (use create, verify, restore)", sub)
	}
}

func cmdServe(ctx context.Context, g globalOpts, opts Options) error {
	bin := dc.DiscoverRPCServer(g.rpcPath)
	if bin == "" {
		return errors.New("deltachat-rpc-server not found")
	}
	tr := dc.NewTransport()
	if err := tr.StartProcess(bin, g.accountsDir); err != nil {
		return err
	}
	defer tr.Close()

	client := dc.NewClient(tr)
	_ = client.StartIOForAllAccounts(ctx)

	ipcSrv, err := dc.StartIPCServer(tr, g.accountsDir)
	if err != nil {
		return err
	}
	defer ipcSrv.Close()

	sigCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		for {
			ev, err := client.GetNextEvent(sigCtx)
			if err != nil {
				return
			}
			ipcSrv.PublishEvent(ev)
		}
	}()

	payload := map[string]any{
		"ok":          true,
		"addr":        ipcSrv.Addr(),
		"accountsDir": g.accountsDir,
		"pid":         os.Getpid(),
	}
	_ = writeOut(opts.Stdout, g.jsonOut, payload, fmt.Sprintf("Parla Viva headless IPC daemon listening on %s (accounts: %s)\n", ipcSrv.Addr(), g.accountsDir))
	<-sigCtx.Done()
	return nil
}


func cmdConfig(args []string, g globalOpts, opts Options) error {
	sub := "show"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		sub = args[0]
		args = args[1:]
	}
	cfg, cfgPath, err := config.Load(g.accountsDir)
	if err != nil {
		return err
	}
	switch sub {
	case "path":
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"path": cfgPath}, cfgPath+"\n")
	case "show":
		raw, _ := json.MarshalIndent(cfg, "", "  ")
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"path": cfgPath, "config": cfg},
			fmt.Sprintf("Config: %s\n%s\n", cfgPath, string(raw)))
	case "init":
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "path": cfgPath, "config": cfg},
			fmt.Sprintf("Initialized config: %s\n", cfgPath))
	case "set":
		if len(args) < 2 {
			return errors.New("config set requires <key> <value>")
		}
		if err := cfg.SetKey(args[0], args[1]); err != nil {
			return err
		}
		if err := config.Save(cfgPath, cfg); err != nil {
			return err
		}
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"ok": true, "path": cfgPath, "key": args[0], "value": args[1], "config": cfg},
			fmt.Sprintf("Updated %s = %s (%s)\n", args[0], args[1], cfgPath))
	default:
		return fmt.Errorf("unknown config subcommand %q (use show, init, set, path)", sub)
	}
}

func cmdLogs(args []string, g globalOpts, opts Options) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	lines := fs.Int("lines", 50, "number of recent log lines")
	filter := fs.String("filter", "", "case-insensitive substring or level filter (e.g. ERROR, WARN)")
	showPath := fs.Bool("path", false, "print log file path only")
	if err := fs.Parse(args); err != nil {
		return err
	}
	cfg, _, _ := config.Load(g.accountsDir)
	if *showPath {
		return writeOut(opts.Stdout, g.jsonOut, map[string]any{"path": cfg.LogFile}, cfg.LogFile+"\n")
	}
	out, err := applog.TailLines(cfg.LogFile, *lines, *filter)
	if err != nil {
		return err
	}
	return writeOut(opts.Stdout, g.jsonOut, map[string]any{
		"path":  cfg.LogFile,
		"count": len(out),
		"lines": out,
	}, strings.Join(out, "\n")+"\n")
}

func printUsage(w io.Writer) {
	fmt.Fprintf(w, `Parla Viva v%s — Desktop Client & Agent CLI

Usage:
  parla-viva [--accounts DIR] [--rpc PATH] [--demo]       Launch desktop GUI (+ local IPC bridge)
  parla-viva <command> [flags]                            Run Agent CLI command

Commands:
  doctor                                                  Check RPC server, IPC bridge & accounts
  accounts [list|create|select]                           List, create Chatmail, or select account
  chats    [list|info|create-group|accept|pin|unpin|mute|unmute|archive]
  messages --chat ID [--limit 30] [--query Q] [--mark-seen]
  send     (--chat ID | --to EMAIL) [--text MSG|-] [--file PATH]... [--reply-to ID]
  react    --msg ID --emoji EMOJI                         Add/clear emoji reaction
  edit     --msg ID --text NEW_TEXT                       Edit sent message
  delete   --msg ID [--msg ID2]                           Delete message(s)
  invite   [--chat ID]                                    Generate SecureJoin invite link
  join     --link URL                                     Join peer/group via SecureJoin link
  watch    [--chat ID] [--incoming] [--once] [--timeout 60s]  Stream live events (NDJSON with --json)
  preview  <filePath>                                     Inspect any common file format
  backup   (create|verify|restore)                        AES-256-GCM SafeStore backup
  config   [show|init|set|path]                           View or edit parla-viva.json
  logs     [--lines 50] [--filter WARN|ERROR] [--path]    Inspect rotating app logs
  serve                                                   Headless IPC daemon for agents
  version                                                 Show version

Global Flags (anywhere):
  --json               Machine-readable JSON / NDJSON output
  --account ID         Target account ID (defaults to active account)
  --accounts DIR       Delta Chat accounts directory
  --rpc PATH           Explicit deltachat-rpc-server binary path
  --direct             Bypass running GUI/daemon IPC bridge
  --timeout DUR        RPC timeout (default 30s)
`, Version)
}
