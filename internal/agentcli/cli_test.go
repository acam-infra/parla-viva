package agentcli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/acam-infra/parla-viva/internal/dc"
)

type mockCaller struct {
	calls []string
}

func (m *mockCaller) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	m.calls = append(m.calls, method)
	switch method {
	case "get_selected_account_id":
		return json.RawMessage(`1`), nil
	case "get_all_accounts":
		return json.RawMessage(`[{"kind":"Configured","id":1,"displayName":"Alice","addr":"alice@example.org"}]`), nil
	case "get_chatlist_entries":
		return json.RawMessage(`[10]`), nil
	case "get_chatlist_items_by_entries":
		return json.RawMessage(`{"10":{"kind":"ChatListItem","name":"Parla Bot","summaryText2":"hi there","freshMessageCounter":2,"isPinned":true}}`), nil
	case "get_message_ids":
		return json.RawMessage(`[101]`), nil
	case "get_messages":
		return json.RawMessage(`{"101":{"kind":"message","id":101,"chatId":10,"text":"hello world","sender":{"displayName":"Parla Bot"},"fromId":2,"timestamp":1732000000,"viewType":"Text"}}`), nil
	case "create_contact":
		return json.RawMessage(`55`), nil
	case "create_chat_by_contact_id":
		return json.RawMessage(`10`), nil
	case "send_msg":
		return json.RawMessage(`202`), nil
	case "send_reaction", "delete_messages", "send_edit_request", "set_chat_visibility", "set_chat_mute_duration", "accept_chat", "markseen_msgs":
		return json.RawMessage(`null`), nil
	case "get_chat_securejoin_qr_code":
		return json.RawMessage(`"https://i.delta.chat/#TESTINVITE"`), nil
	case "secure_join":
		return json.RawMessage(`15`), nil
	default:
		return json.RawMessage(`null`), nil
	}
}

func TestShouldHandleRouting(t *testing.T) {
	if ShouldHandle(nil) {
		t.Fatal("empty args should launch GUI")
	}
	if ShouldHandle([]string{"--accounts", "/tmp/a", "--demo"}) {
		t.Fatal("GUI flags should launch GUI")
	}
	for _, tc := range [][]string{
		{"send", "--chat", "10", "--text", "hi"},
		{"--json", "chats", "list"},
		{"--accounts", "/tmp/a", "messages", "--chat", "10"},
		{"cli", "accounts", "list"},
		{"version"},
	} {
		if !ShouldHandle(tc) {
			t.Fatalf("expected ShouldHandle(%v) = true", tc)
		}
	}
}

func TestAgentCLICommandsJSON(t *testing.T) {
	mc := &mockCaller{}
	client := dc.NewClientWithCaller(mc)

	run := func(stdin string, args ...string) (int, string, string) {
		var out, errBuf bytes.Buffer
		code := Run(context.Background(), args, Options{
			Stdin:  strings.NewReader(stdin),
			Stdout: &out,
			Stderr: &errBuf,
			Client: client,
		})
		return code, out.String(), errBuf.String()
	}

	// 1. accounts list --json
	code, out, errOut := run("", "accounts", "list", "--json")
	if code != 0 || !strings.Contains(out, `"alice@example.org"`) {
		t.Fatalf("accounts list failed: code=%d out=%s err=%s", code, out, errOut)
	}

	// 2. chats list --json
	code, out, errOut = run("", "chats", "list", "--json")
	if code != 0 || !strings.Contains(out, `"Parla Bot"`) {
		t.Fatalf("chats list failed: code=%d out=%s err=%s", code, out, errOut)
	}

	// 3. send via stdin + --to email + multi-file
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "a.txt")
	f2 := filepath.Join(tmp, "b.json")
	_ = os.WriteFile(f1, []byte("hello"), 0o600)
	_ = os.WriteFile(f2, []byte(`{"ok":true}`), 0o600)

	code, out, errOut = run("stdin caption\n", "send", "--to", "bot@nine.testrun.org", "--text", "-", "--file", f1, "--file", f2, "--json")
	if code != 0 || !strings.Contains(out, `"chatId": 10`) || !strings.Contains(out, `202`) {
		t.Fatalf("send failed: code=%d out=%s err=%s", code, out, errOut)
	}

	// 4. preview JSON file
	code, out, errOut = run("", "preview", f2, "--json")
	if code != 0 || !strings.Contains(out, `"JSON"`) {
		t.Fatalf("preview failed: code=%d out=%s err=%s", code, out, errOut)
	}

	// 5. invite & join
	code, out, _ = run("", "invite", "--json")
	if code != 0 || !strings.Contains(out, "https://i.delta.chat/#TESTINVITE") {
		t.Fatalf("invite failed: code=%d out=%s", code, out)
	}
	code, out, _ = run("", "join", "--link", "https://i.delta.chat/#TESTINVITE", "--json")
	if code != 0 || !strings.Contains(out, `"chatId": 15`) {
		t.Fatalf("join failed: code=%d out=%s", code, out)
	}

	// 6. watch --incoming --once --json over live IPC bridge
	ipcDir := t.TempDir()
	srv, err := dc.StartIPCServer(mc, ipcDir)
	if err != nil {
		t.Fatalf("StartIPCServer: %v", err)
	}
	defer srv.Close()

	ipcClient, err := dc.DiscoverIPC(context.Background(), ipcDir)
	if err != nil || ipcClient == nil {
		t.Fatalf("DiscoverIPC: %v", err)
	}
	var watchOut, watchErr bytes.Buffer
	done := make(chan int, 1)
	go func() {
		done <- Run(context.Background(), []string{"watch", "--incoming", "--once", "--timeout", "3s", "--json"}, Options{
			Stdout: &watchOut,
			Stderr: &watchErr,
			Client: dc.NewClientWithCaller(ipcClient),
			IPC:    ipcClient,
		})
	}()
	time.Sleep(100 * time.Millisecond)
	srv.PublishEvent(dc.Event{
		ContextID: 1,
		Event:     dc.EventPayload{Kind: "IncomingMsg", ChatID: 10, MsgID: 101},
	})
	if c := <-done; c != 0 {
		t.Fatalf("watch failed: code=%d out=%s err=%s", c, watchOut.String(), watchErr.String())
	}
	if !strings.Contains(watchOut.String(), `"hello world"`) {
		t.Fatalf("expected hydrated message in watch output, got: %s", watchOut.String())
	}
}
