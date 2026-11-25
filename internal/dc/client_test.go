package dc

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"
)

func startFakeRPCServer(t *testing.T, handler func(method string, params []json.RawMessage) (any, *rpcError)) *Client {
	t.Helper()
	clientToSrvR, clientToSrvW := io.Pipe()
	srvToClientR, srvToClientW := io.Pipe()

	go func() {
		defer srvToClientW.Close()
		scanner := bufio.NewScanner(clientToSrvR)
		for scanner.Scan() {
			var req struct {
				ID     int64             `json:"id"`
				Method string            `json:"method"`
				Params []json.RawMessage `json:"params"`
			}
			if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
				continue
			}
			res, rpcErr := handler(req.Method, req.Params)
			resp := map[string]any{
				"jsonrpc": "2.0",
				"id":      req.ID,
			}
			if rpcErr != nil {
				resp["error"] = rpcErr
			} else {
				resp["result"] = res
			}
			b, _ := json.Marshal(resp)
			_, _ = srvToClientW.Write(append(b, '\n'))
		}
	}()

	tr := NewTransport()
	tr.StartPipes(srvToClientR, clientToSrvW)
	t.Cleanup(func() {
		_ = tr.Close()
		_ = clientToSrvR.Close()
	})
	return NewClient(tr)
}

func TestClientAccountsChatsAndMessages(t *testing.T) {
	client := startFakeRPCServer(t, func(method string, params []json.RawMessage) (any, *rpcError) {
		switch method {
		case "get_all_accounts":
			return []map[string]any{
				{"id": 1, "kind": "Configured", "displayName": "Alice", "addr": "alice@nine.testrun.org"},
			}, nil
		case "select_account":
			return nil, nil
		case "get_chatlist_entries":
			return []int{10, 11}, nil
		case "get_chatlist_items_by_entries":
			return map[string]any{
				"10": map[string]any{
					"kind": "ChatListItem", "name": "Viva Core Team", "chatType": "Group",
					"summaryText1": "Bob", "summaryText2": "Shipped v0.6.6!", "freshMessageCounter": 2,
					"lastUpdated": int64(1763600000000), "isPinned": true, "freshness": "RecentlySeen",
				},
				"11": map[string]any{
					"kind": "ChatListItem", "name": "Saved Messages", "chatType": "Single",
					"isSelfTalk": true, "summaryText2": "Release notes", "lastUpdated": int64(1763590000000),
				},
			}, nil
		case "get_messages":
			return map[string]any{
				"101": map[string]any{
					"kind": "message", "id": 101, "chatId": 10, "fromId": 1,
					"text": "Migrating Parla to Viva!", "timestamp": int64(1763600000),
					"state": int(MsgStateOutDelivered), "viewType": "Text",
					"sender": map[string]any{"id": 1, "displayName": "Alice"},
					"reactions": map[string]any{
						"reactions": []map[string]any{{"emoji": "🔥", "count": 3, "isFromSelf": true}},
					},
				},
			}, nil
		case "send_msg":
			return 102, nil
		case "get_next_event":
			return map[string]any{
				"contextId": 1,
				"event":     map[string]any{"kind": "IncomingMsg", "chatId": 10, "msgId": 103},
			}, nil
		case "get_contacts":
			return []map[string]any{
				{
					"id": 2, "displayName": "Bob", "address": "bob@nine.testrun.org",
					"isVerified": true, "wasSeenRecently": true, "status": "Building Viva desktop UI",
				},
			}, nil
		case "get_chat_contacts":
			return []int{1, 2}, nil
		default:
			return nil, &rpcError{Code: -32601, Message: "method not found: " + method}
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	accounts, err := client.GetAllAccounts(ctx)
	if err != nil || len(accounts) != 1 || accounts[0].DisplayName != "Alice" {
		t.Fatalf("GetAllAccounts = %+v, err=%v", accounts, err)
	}
	if err := client.SelectAccount(ctx, 1); err != nil {
		t.Fatalf("SelectAccount: %v", err)
	}

	entries, err := client.GetChatlistEntries(ctx, 1, nil, "")
	if err != nil || len(entries) != 2 {
		t.Fatalf("GetChatlistEntries = %v, err=%v", entries, err)
	}

	chats, err := client.GetChatlistItems(ctx, 1, entries)
	if err != nil || len(chats) != 2 {
		t.Fatalf("GetChatlistItems = %+v, err=%v", chats, err)
	}
	if chats[0].Kind != ChatGroup || !chats[0].IsPinned || !chats[0].WasSeenRecently || chats[0].UnreadCount != 2 {
		t.Fatalf("unexpected first chat: %+v", chats[0])
	}
	if chats[1].Kind != ChatSavedMessages {
		t.Fatalf("expected ChatSavedMessages, got %v", chats[1].Kind)
	}

	msgs, err := client.GetMessages(ctx, 1, []int{101})
	if err != nil || len(msgs) != 1 {
		t.Fatalf("GetMessages = %+v, err=%v", msgs, err)
	}
	if !msgs[0].IsOutgoing || msgs[0].StatusLabel() != "Delivered" || len(msgs[0].Reactions) != 1 {
		t.Fatalf("unexpected message: %+v", msgs[0])
	}

	sentID, err := client.SendTextMessage(ctx, 1, 10, "Hello Viva", 101)
	if err != nil || sentID != 102 {
		t.Fatalf("SendTextMessage = %d, err=%v", sentID, err)
	}

	ev, err := client.GetNextEvent(ctx)
	if err != nil || ev.Event.Kind != "IncomingMsg" || ev.Event.MsgID != 103 {
		t.Fatalf("GetNextEvent = %+v, err=%v", ev, err)
	}

	contacts, err := client.GetContacts(ctx, 1, "")
	if err != nil || len(contacts) != 1 || contacts[0].DisplayLabel() != "Bob" || !contacts[0].IsVerified {
		t.Fatalf("GetContacts = %+v, err=%v", contacts, err)
	}

	memberIDs, err := client.GetChatContacts(ctx, 1, 10)
	if err != nil || len(memberIDs) != 2 {
		t.Fatalf("GetChatContacts = %v, err=%v", memberIDs, err)
	}
}
