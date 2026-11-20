package dc

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
)

// Client provides typed Delta Chat JSON-RPC methods on top of a Caller
// (either a direct *Transport or a local *IPCClient).
type Client struct {
	caller    Caller
	transport *Transport
	accountID int
}

// NewClient wraps an existing direct Transport.
func NewClient(transport *Transport) *Client {
	return &Client{caller: transport, transport: transport}
}

// NewClientWithCaller wraps any Caller implementation (such as *IPCClient).
func NewClientWithCaller(caller Caller) *Client {
	t, _ := caller.(*Transport)
	return &Client{caller: caller, transport: t}
}

// Caller exposes the underlying JSON-RPC caller.
func (c *Client) Caller() Caller {
	return c.caller
}

// Transport exposes the underlying direct stdio transport (nil when using IPC).
func (c *Client) Transport() *Transport {
	return c.transport
}

// AccountID returns the currently selected account ID.
func (c *Client) AccountID() int {
	return c.accountID
}

// SetAccountID updates the active account ID tracked by the client.
func (c *Client) SetAccountID(id int) {
	c.accountID = id
}

// GetAllAccounts returns all accounts known to deltachat-rpc-server.
func (c *Client) GetAllAccounts(ctx context.Context) ([]Account, error) {
	raw, err := c.caller.Call(ctx, "get_all_accounts", []any{})
	if err != nil {
		return nil, err
	}
	var accounts []Account
	if err := json.Unmarshal(raw, &accounts); err != nil {
		return nil, fmt.Errorf("dc: decode accounts: %w", err)
	}
	return accounts, nil
}

// SelectAccount selects the active account on the RPC server.
func (c *Client) SelectAccount(ctx context.Context, accountID int) error {
	if _, err := c.caller.Call(ctx, "select_account", []any{accountID}); err != nil {
		return err
	}
	c.accountID = accountID
	return nil
}

// StartIOForAllAccounts starts network I/O across all configured accounts.
func (c *Client) StartIOForAllAccounts(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "start_io_for_all_accounts", []any{})
	return err
}

// GetChatlistEntries fetches ordered chat IDs for an account, optionally filtered by query.
func (c *Client) GetChatlistEntries(ctx context.Context, accountID int, listFlags *int, query string) ([]int, error) {
	var q any
	if query != "" {
		q = query
	}
	raw, err := c.caller.Call(ctx, "get_chatlist_entries", []any{accountID, listFlags, q, nil})
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("dc: decode chatlist entries: %w", err)
	}
	return ids, nil
}

// GetChatlistItems fetches hydrated chatlist entries in the exact order of chatIDs.
func (c *Client) GetChatlistItems(ctx context.Context, accountID int, chatIDs []int) ([]ChatEntry, error) {
	if len(chatIDs) == 0 {
		return nil, nil
	}
	raw, err := c.caller.Call(ctx, "get_chatlist_items_by_entries", []any{accountID, chatIDs})
	if err != nil {
		return nil, err
	}
	var itemsMap map[string]rawChatItem
	if err := json.Unmarshal(raw, &itemsMap); err != nil {
		return nil, fmt.Errorf("dc: decode chatlist items: %w", err)
	}
	out := make([]ChatEntry, 0, len(chatIDs))
	for _, id := range chatIDs {
		item, ok := itemsMap[strconv.Itoa(id)]
		if !ok || item.Kind == "Error" || item.Kind == "ArchiveLink" {
			continue
		}
		out = append(out, item.toChatEntry(id))
	}
	return out, nil
}

// GetMessageIDs returns all message IDs for a chat in chronological order.
func (c *Client) GetMessageIDs(ctx context.Context, accountID, chatID int) ([]int, error) {
	raw, err := c.caller.Call(ctx, "get_message_ids", []any{accountID, chatID, false, false})
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("dc: decode message ids: %w", err)
	}
	return ids, nil
}

// GetMessages fetches and parses a batch of messages in the order of msgIDs.
func (c *Client) GetMessages(ctx context.Context, accountID int, msgIDs []int) ([]Message, error) {
	if len(msgIDs) == 0 {
		return nil, nil
	}
	raw, err := c.caller.Call(ctx, "get_messages", []any{accountID, msgIDs})
	if err != nil {
		return nil, err
	}
	var msgMap map[string]rawMessageEnvelope
	if err := json.Unmarshal(raw, &msgMap); err != nil {
		return nil, fmt.Errorf("dc: decode messages: %w", err)
	}
	out := make([]Message, 0, len(msgIDs))
	for _, id := range msgIDs {
		env, ok := msgMap[strconv.Itoa(id)]
		if !ok || env.Kind == "loadingError" {
			continue
		}
		out = append(out, env.toMessage())
	}
	return out, nil
}

// SendTextMessage sends a plain-text (or Markdown) message to a chat.
func (c *Client) SendTextMessage(ctx context.Context, accountID, chatID int, text string, quotedMsgID int) (int, error) {
	data := map[string]any{
		"text": text,
	}
	if quotedMsgID > 0 {
		data["quotedMessageId"] = quotedMsgID
	}
	raw, err := c.caller.Call(ctx, "send_msg", []any{accountID, chatID, data})
	if err != nil {
		return 0, err
	}
	var msgID int
	if err := json.Unmarshal(raw, &msgID); err != nil {
		return 0, fmt.Errorf("dc: decode sent msg id: %w", err)
	}
	return msgID, nil
}

// SendReaction adds or replaces the user's emoji reaction on a message.
func (c *Client) SendReaction(ctx context.Context, accountID, msgID int, emojis ...string) error {
	_, err := c.caller.Call(ctx, "send_reaction", []any{accountID, msgID, emojis})
	return err
}

// ResendMessages retries failed outgoing messages (mirrors Parla CORE-003).
func (c *Client) ResendMessages(ctx context.Context, accountID int, msgIDs []int) error {
	_, err := c.caller.Call(ctx, "resend_messages", []any{accountID, msgIDs})
	return err
}

// MarkSeenMsgs marks visible message IDs as read.
func (c *Client) MarkSeenMsgs(ctx context.Context, accountID int, msgIDs []int) error {
	if len(msgIDs) == 0 {
		return nil
	}
	_, err := c.caller.Call(ctx, "markseen_msgs", []any{accountID, msgIDs})
	return err
}

// SearchMessages searches messages within a chat (chatID > 0) or across all chats (chatID == 0).
func (c *Client) SearchMessages(ctx context.Context, accountID int, chatID int, query string) ([]int, error) {
	var cid any
	if chatID > 0 {
		cid = chatID
	}
	raw, err := c.caller.Call(ctx, "search_messages", []any{accountID, query, cid})
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("dc: decode search ids: %w", err)
	}
	return ids, nil
}

// GetNextEvent blocks until the next core event arrives.
func (c *Client) GetNextEvent(ctx context.Context) (Event, error) {
	raw, err := c.caller.Call(ctx, "get_next_event", []any{})
	if err != nil {
		return Event{}, err
	}
	var ev Event
	if err := json.Unmarshal(raw, &ev); err != nil {
		return Event{}, fmt.Errorf("dc: decode event: %w", err)
	}
	return ev, nil
}

// GetWebxdcBlob fetches a file from a Webxdc attachment via core RPC.
func (c *Client) GetWebxdcBlob(ctx context.Context, accountID, msgID int, path string) ([]byte, error) {
	raw, err := c.caller.Call(ctx, "get_webxdc_blob", []any{accountID, msgID, path})
	if err != nil {
		return nil, err
	}
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		return nil, fmt.Errorf("dc: decode webxdc blob: %w", err)
	}
	return base64.StdEncoding.DecodeString(encoded)
}

type rawChatItem struct {
	Kind                string `json:"kind"`
	Name                string `json:"name"`
	SummaryText1        string `json:"summaryText1"`
	SummaryText2        string `json:"summaryText2"`
	LastMessageID       int    `json:"lastMessageId"`
	SummaryStatus       int    `json:"summaryStatus"`
	FreshMessageCounter int    `json:"freshMessageCounter"`
	LastUpdated         int64  `json:"lastUpdated"`
	AvatarPath          string `json:"avatarPath"`
	ChatType            string `json:"chatType"`
	IsSelfTalk          bool   `json:"isSelfTalk"`
	IsDeviceChat        bool   `json:"isDeviceChat"`
	IsMuted             bool   `json:"isMuted"`
	IsPinned            bool   `json:"isPinned"`
	IsArchived          bool   `json:"isArchived"`
	IsContactRequest    bool   `json:"isContactRequest"`
	Freshness           string `json:"freshness"`
	WasSeenRecently     bool   `json:"wasSeenRecently"`
}

func (r rawChatItem) toChatEntry(id int) ChatEntry {
	kind := ChatDirect
	switch {
	case r.IsSelfTalk:
		kind = ChatSavedMessages
	case r.IsDeviceChat:
		kind = ChatDevice
	case r.ChatType == "Group":
		kind = ChatGroup
	case r.ChatType == "Broadcast" || r.ChatType == "OutBroadcast" || r.ChatType == "InBroadcast":
		kind = ChatChannel
	}
	seenRecently := r.WasSeenRecently
	if r.Freshness != "" {
		seenRecently = r.Freshness == "RecentlySeen"
	}
	return ChatEntry{
		ID:               id,
		Name:             r.Name,
		SummaryPrefix:    r.SummaryText1,
		LastMessage:      r.SummaryText2,
		LastMessageID:    r.LastMessageID,
		UnreadCount:      r.FreshMessageCounter,
		Timestamp:        r.LastUpdated / 1000,
		AvatarPath:       r.AvatarPath,
		Kind:             kind,
		ChatType:         r.ChatType,
		IsMuted:          r.IsMuted,
		IsPinned:         r.IsPinned,
		IsArchived:       r.IsArchived,
		IsContactRequest: r.IsContactRequest,
		IsDraft:          MessageState(r.SummaryStatus) == MsgStateOutDraft,
		WasSeenRecently:  seenRecently,
	}
}

type rawMessageEnvelope struct {
	Kind               string `json:"kind"`
	ID                 int    `json:"id"`
	ChatID             int    `json:"chatId"`
	FromID             int    `json:"fromId"`
	Text               string `json:"text"`
	Timestamp          int64  `json:"timestamp"`
	IsInfo             bool   `json:"isInfo"`
	IsForwarded        bool   `json:"isForwarded"`
	IsEdited           bool   `json:"isEdited"`
	IsPinned           bool   `json:"isPinned"`
	OverrideSenderName string `json:"overrideSenderName"`
	File               string `json:"file"`
	FileName           string `json:"fileName"`
	FileMime           string `json:"fileMime"`
	FileBytes          int64  `json:"fileBytes"`
	ViewType           string `json:"viewType"`
	State              int    `json:"state"`
	Error              string `json:"error"`
	HasHTML            bool   `json:"hasHtml"`
	DownloadState      string `json:"downloadState"`
	Sender             *struct {
		ID          int    `json:"id"`
		DisplayName string `json:"displayName"`
		Name        string `json:"name"`
		Address     string `json:"address"`
	} `json:"sender"`
	Quote *struct {
		Kind              string `json:"kind"`
		MessageID         int    `json:"messageId"`
		AuthorDisplayName string `json:"authorDisplayName"`
		Text              string `json:"text"`
	} `json:"quote"`
	Reactions *struct {
		Reactions []struct {
			Emoji      string `json:"emoji"`
			Count      int    `json:"count"`
			IsFromSelf bool   `json:"isFromSelf"`
		} `json:"reactions"`
	} `json:"reactions"`
}

func (r rawMessageEnvelope) toMessage() Message {
	m := Message{
		ID:                 r.ID,
		ChatID:             r.ChatID,
		Text:               r.Text,
		Timestamp:          r.Timestamp,
		IsInfo:             r.IsInfo,
		IsForwarded:        r.IsForwarded,
		IsEdited:           r.IsEdited,
		IsPinned:           r.IsPinned,
		OverrideSenderName: r.OverrideSenderName,
		FilePath:           r.File,
		FileName:           r.FileName,
		FileMime:           r.FileMime,
		FileBytes:          r.FileBytes,
		ViewType:           r.ViewType,
		State:              MessageState(r.State),
		Error:              r.Error,
		HasHTML:            r.HasHTML,
		DownloadState:      r.DownloadState,
		SenderContactID:    r.FromID,
	}
	if r.Sender != nil {
		if m.SenderContactID == 0 {
			m.SenderContactID = r.Sender.ID
		}
		m.SenderAddress = r.Sender.Address
		if r.Sender.DisplayName != "" {
			m.SenderName = r.Sender.DisplayName
		} else {
			m.SenderName = r.Sender.Name
		}
	}
	m.IsOutgoing = m.SenderContactID == 1

	if r.Quote != nil && r.Quote.Text != "" {
		m.Quote = &Quote{
			MessageID:  r.Quote.MessageID,
			AuthorName: r.Quote.AuthorDisplayName,
			Text:       r.Quote.Text,
		}
	}
	if r.Reactions != nil {
		for _, rx := range r.Reactions.Reactions {
			m.Reactions = append(m.Reactions, Reaction{
				Emoji:  rx.Emoji,
				Count:  rx.Count,
				BySelf: rx.IsFromSelf,
			})
		}
	}
	return m
}
