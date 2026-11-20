package appui

import (
	"testing"

	"github.com/acam-infra/parla-viva/internal/dc"
)

func TestDiskCacheInstantFirstPaintAndDeltaMerge(t *testing.T) {
	dir := t.TempDir()

	m1 := NewLiveModel(nil, nil)
	m1.AccountsPath = dir
	m1.Accounts = []dc.Account{{ID: 1, Kind: "Configured", DisplayName: "ccds"}}
	m1.ActiveAcctID = 1
	m1.ActiveChatID = 12
	m1.Chats = []dc.ChatEntry{
		{ID: 12, Name: "Mac", LastMessage: "hello", LastMessageID: 101},
		{ID: 15, Name: "Parla Viva Bot", LastMessage: "ping", LastMessageID: 201},
	}
	m1.Messages[12] = []dc.Message{
		{ID: 100, ChatID: 12, Text: "first", Timestamp: 1000},
		{ID: 101, ChatID: 12, Text: "hello", Timestamp: 1005, State: dc.MsgStateOutPending},
	}

	if err := m1.SaveCache(); err != nil {
		t.Fatalf("SaveCache: %v", err)
	}

	// Cold start: LoadCache hydrates Frame 1 synchronously before RPC connects
	m2 := NewLiveModel(nil, nil)
	m2.AccountsPath = dir
	if !m2.LoadCache(dir) {
		t.Fatal("expected LoadCache to return true")
	}
	if m2.ActiveChatID != 12 || len(m2.Chats) != 2 || len(m2.Messages[12]) != 2 {
		t.Fatalf("unexpected cached state: active=%d chats=%d msgs=%d", m2.ActiveChatID, len(m2.Chats), len(m2.Messages[12]))
	}

	// Incremental delta merge: update state of #101 + append #102
	merged := mergeDeltaMessages(m2.Messages[12], []dc.Message{
		{ID: 101, ChatID: 12, Text: "hello", Timestamp: 1005, State: dc.MsgStateOutDelivered},
		{ID: 102, ChatID: 12, Text: "reply", Timestamp: 1010},
	})
	if len(merged) != 3 || merged[1].State != dc.MsgStateOutDelivered || merged[2].ID != 102 {
		t.Fatalf("unexpected delta merge result: %+v", merged)
	}
}
