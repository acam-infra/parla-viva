package appui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/history"
)

const (
	cacheFilename      = ".parla-viva-cache.json"
	cacheMaxChats      = 10
	cacheMaxMsgsPerKey = 40
)

type diskSnapshot struct {
	Version      int                  `json:"version"`
	SavedAt      int64                `json:"savedAt"`
	Accounts     []dc.Account         `json:"accounts"`
	ActiveAcctID int                  `json:"activeAcctId"`
	ActiveChatID int                  `json:"activeChatId"`
	InviteLink   string               `json:"inviteLink,omitempty"`
	Chats        []dc.ChatEntry       `json:"chats"`
	Messages     map[int][]dc.Message `json:"messages"`
}

func cachePathFor(accountsDir string) string {
	if accountsDir == "" {
		return ""
	}
	return filepath.Join(accountsDir, cacheFilename)
}

// LoadCache synchronously hydrates Accounts, Chats, ActiveChatID, and recent
// Messages from the local disk snapshot so Frame 1 renders immediately with zero
// blank screen while live RPC delta sync catches up in the background.
func (m *Model) LoadCache(accountsDir string) bool {
	p := cachePathFor(accountsDir)
	if p == "" {
		return false
	}
	raw, err := os.ReadFile(p)
	if err != nil {
		return false
	}
	var snap diskSnapshot
	if err := json.Unmarshal(raw, &snap); err != nil || snap.Version != 1 {
		return false
	}
	if len(snap.Accounts) > 0 {
		m.Accounts = snap.Accounts
	}
	if snap.ActiveAcctID > 0 {
		m.ActiveAcctID = snap.ActiveAcctID
	}
	if len(snap.Chats) > 0 {
		m.Chats = snap.Chats
	}
	if snap.InviteLink != "" {
		m.InviteLink = snap.InviteLink
	}
	for chatID, msgs := range snap.Messages {
		if len(msgs) == 0 {
			continue
		}
		m.Messages[chatID] = msgs
		ids := make([]int, len(msgs))
		for i, msg := range msgs {
			ids[i] = msg.ID
		}
		m.Windows[chatID] = history.NewLatest(ids, 0, history.DefaultBatchSize)
	}
	if snap.ActiveChatID > 0 {
		m.ActiveChatID = snap.ActiveChatID
	} else if len(m.Chats) > 0 {
		m.ActiveChatID = m.Chats[0].ID
	}
	return len(m.Chats) > 0
}

// SaveCache atomically writes a bounded snapshot of current accounts, chats,
// and recent messages for instant first-paint on the next launch.
func (m *Model) SaveCache() error {
	if m.DemoMode || m.AccountsPath == "" {
		return nil
	}
	p := cachePathFor(m.AccountsPath)
	if p == "" {
		return nil
	}

	snap := diskSnapshot{
		Version:      1,
		SavedAt:      time.Now().Unix(),
		Accounts:     append([]dc.Account(nil), m.Accounts...),
		ActiveAcctID: m.ActiveAcctID,
		ActiveChatID: m.ActiveChatID,
		InviteLink:   m.InviteLink,
		Chats:        append([]dc.ChatEntry(nil), m.Chats...),
		Messages:     make(map[int][]dc.Message),
	}

	// Keep recent messages for the active chat plus top chats only so startup load stays <5ms.
	keep := make(map[int]bool)
	if m.ActiveChatID > 0 {
		keep[m.ActiveChatID] = true
	}
	for i := 0; i < len(m.Chats) && len(keep) < cacheMaxChats; i++ {
		keep[m.Chats[i].ID] = true
	}
	for chatID := range keep {
		msgs := m.Messages[chatID]
		if len(msgs) == 0 {
			continue
		}
		if len(msgs) > cacheMaxMsgsPerKey {
			msgs = msgs[len(msgs)-cacheMaxMsgsPerKey:]
		}
		cp := make([]dc.Message, len(msgs))
		copy(cp, msgs)
		snap.Messages[chatID] = cp
	}

	raw, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

func (m *Model) saveCacheAsync() {
	if m.DemoMode || m.AccountsPath == "" {
		return
	}
	// Capture lightweight snapshot on UI thread, write on background goroutine.
	accts := append([]dc.Account(nil), m.Accounts...)
	chats := append([]dc.ChatEntry(nil), m.Chats...)
	activeAcct := m.ActiveAcctID
	activeChat := m.ActiveChatID
	invite := m.InviteLink
	path := cachePathFor(m.AccountsPath)

	msgsCopy := make(map[int][]dc.Message)
	keep := make(map[int]bool)
	if activeChat > 0 {
		keep[activeChat] = true
	}
	for i := 0; i < len(chats) && len(keep) < cacheMaxChats; i++ {
		keep[chats[i].ID] = true
	}
	for cid := range keep {
		msgs := m.Messages[cid]
		if len(msgs) == 0 {
			continue
		}
		if len(msgs) > cacheMaxMsgsPerKey {
			msgs = msgs[len(msgs)-cacheMaxMsgsPerKey:]
		}
		cp := make([]dc.Message, len(msgs))
		copy(cp, msgs)
		msgsCopy[cid] = cp
	}

	go func() {
		snap := diskSnapshot{
			Version:      1,
			SavedAt:      time.Now().Unix(),
			Accounts:     accts,
			ActiveAcctID: activeAcct,
			ActiveChatID: activeChat,
			InviteLink:   invite,
			Chats:        chats,
			Messages:     msgsCopy,
		}
		raw, err := json.Marshal(snap)
		if err != nil {
			return
		}
		_ = os.MkdirAll(filepath.Dir(path), 0o700)
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, raw, 0o600); err == nil {
			_ = os.Rename(tmp, path)
		}
	}()
}

// mergeDeltaMessages applies incremental message updates (new messages, read-state
// updates, edits, or reactions) onto an existing sorted slice without full replacement.
func mergeDeltaMessages(existing, delta []dc.Message) []dc.Message {
	if len(delta) == 0 {
		return existing
	}
	byID := make(map[int]dc.Message, len(existing)+len(delta))
	for _, msg := range existing {
		byID[msg.ID] = msg
	}
	for _, msg := range delta {
		byID[msg.ID] = msg
	}
	out := make([]dc.Message, 0, len(byID))
	for _, msg := range byID {
		out = append(out, msg)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Timestamp != out[j].Timestamp {
			return out[i].Timestamp < out[j].Timestamp
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// upsertChatEntry updates a single ChatEntry and optionally bubbles it to the top
// (respecting pinned chats) so incoming messages surface immediately in the sidebar.
func upsertChatEntry(chats []dc.ChatEntry, updated dc.ChatEntry, bubbleTop bool) []dc.ChatEntry {
	rest := make([]dc.ChatEntry, 0, len(chats))
	for _, ch := range chats {
		if ch.ID != updated.ID {
			rest = append(rest, ch)
		}
	}
	if !bubbleTop {
		// Preserve existing index if present.
		out := make([]dc.ChatEntry, 0, len(chats)+1)
		found := false
		for _, ch := range chats {
			if ch.ID == updated.ID {
				out = append(out, updated)
				found = true
			} else {
				out = append(out, ch)
			}
		}
		if !found {
			out = append([]dc.ChatEntry{updated}, out...)
		}
		return out
	}
	// Insert right after pinned chats (or at index 0 if updated itself is pinned).
	insertAt := 0
	if !updated.IsPinned {
		for insertAt < len(rest) && rest[insertAt].IsPinned {
			insertAt++
		}
	}
	out := make([]dc.ChatEntry, 0, len(rest)+1)
	out = append(out, rest[:insertAt]...)
	out = append(out, updated)
	out = append(out, rest[insertAt:]...)
	return out
}
