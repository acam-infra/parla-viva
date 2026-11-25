package dc

import (
	"context"
	"encoding/json"
	"fmt"
)

// AddAccount creates a new unconfigured profile on deltachat-rpc-server.
func (c *Client) AddAccount(ctx context.Context) (int, error) {
	raw, err := c.caller.Call(ctx, "add_account", []any{})
	if err != nil {
		return 0, err
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("dc: decode new account id: %w", err)
	}
	return id, nil
}

// GetSelectedAccountID returns the currently selected account ID on the RPC server, or 0 if none.
func (c *Client) GetSelectedAccountID(ctx context.Context) (int, error) {
	raw, err := c.caller.Call(ctx, "get_selected_account_id", []any{})
	if err != nil {
		return 0, err
	}
	if string(raw) == "null" {
		return 0, nil
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("dc: decode selected account id: %w", err)
	}
	return id, nil
}

// GetConnectivity returns the numeric connectivity status (4000=connected, 3000=syncing, etc.).
func (c *Client) GetConnectivity(ctx context.Context, accountID int) (int, error) {
	raw, err := c.caller.Call(ctx, "get_connectivity", []any{accountID})
	if err != nil {
		return 0, err
	}
	var state int
	if err := json.Unmarshal(raw, &state); err != nil {
		return 0, fmt.Errorf("dc: decode connectivity: %w", err)
	}
	return state, nil
}

// SendFileMessage sends a message with a local file attachment and optional caption.
func (c *Client) SendFileMessage(ctx context.Context, accountID, chatID int, filePath, caption string) (int, error) {
	return c.SendFileMessageWithQuote(ctx, accountID, chatID, filePath, caption, 0)
}

// SendFileMessageWithQuote sends a file attachment with optional caption and reply quote.
func (c *Client) SendFileMessageWithQuote(ctx context.Context, accountID, chatID int, filePath, caption string, quotedMsgID int) (int, error) {
	data := map[string]any{
		"file": filePath,
	}
	if caption != "" {
		data["text"] = caption
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
		return 0, fmt.Errorf("dc: decode sent file msg id: %w", err)
	}
	return msgID, nil
}

// CreateContact creates or looks up a contact by email address.
func (c *Client) CreateContact(ctx context.Context, accountID int, email, name string) (int, error) {
	var n any
	if name != "" {
		n = name
	}
	raw, err := c.caller.Call(ctx, "create_contact", []any{accountID, email, n})
	if err != nil {
		return 0, err
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("dc: decode contact id: %w", err)
	}
	return id, nil
}

// CreateChatByContactID creates or opens the 1:1 chat with a contact ID.
func (c *Client) CreateChatByContactID(ctx context.Context, accountID, contactID int) (int, error) {
	raw, err := c.caller.Call(ctx, "create_chat_by_contact_id", []any{accountID, contactID})
	if err != nil {
		return 0, err
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("dc: decode chat id: %w", err)
	}
	return id, nil
}

// CreateGroupChat creates a new group chat and returns its chat ID.
func (c *Client) CreateGroupChat(ctx context.Context, accountID int, name string, protect bool) (int, error) {
	raw, err := c.caller.Call(ctx, "create_group_chat", []any{accountID, name, protect})
	if err != nil {
		return 0, err
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("dc: decode group chat id: %w", err)
	}
	return id, nil
}

// AcceptChat accepts a contact request chat.
func (c *Client) AcceptChat(ctx context.Context, accountID, chatID int) error {
	_, err := c.caller.Call(ctx, "accept_chat", []any{accountID, chatID})
	return err
}

// BasicChatInfo holds metadata returned by get_basic_chat_info.
type BasicChatInfo struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	ChatType      string `json:"chatType"`
	Archived      bool   `json:"archived"`
	Pinned        bool   `json:"pinned"`
	IsMuted       bool   `json:"isMuted"`
	IsContactReq  bool   `json:"isContactRequest"`
	IsSelfTalk    bool   `json:"isSelfTalk"`
	IsDeviceChat  bool   `json:"isDeviceChat"`
	IsEncrypted   bool   `json:"isEncrypted"`
	Color         string `json:"color"`
	ProfileImage  string `json:"profileImage"`
}

// GetBasicChatInfo fetches detailed metadata for a single chat.
func (c *Client) GetBasicChatInfo(ctx context.Context, accountID, chatID int) (BasicChatInfo, error) {
	raw, err := c.caller.Call(ctx, "get_basic_chat_info", []any{accountID, chatID})
	if err != nil {
		return BasicChatInfo{}, err
	}
	var info BasicChatInfo
	if err := json.Unmarshal(raw, &info); err != nil {
		return BasicChatInfo{}, fmt.Errorf("dc: decode basic chat info: %w", err)
	}
	return info, nil
}

// DeleteMessages deletes one or more messages locally (Delete for Me).
func (c *Client) DeleteMessages(ctx context.Context, accountID int, msgIDs []int) error {
	if len(msgIDs) == 0 {
		return nil
	}
	_, err := c.caller.Call(ctx, "delete_messages", []any{accountID, msgIDs})
	return err
}

// SendEditRequest edits a previously sent outgoing message.
func (c *Client) SendEditRequest(ctx context.Context, accountID, msgID int, newText string) error {
	_, err := c.caller.Call(ctx, "send_edit_request", []any{accountID, msgID, newText})
	return err
}

// SetChatVisibility toggles pinned ("Pinned"), archived ("Archived"), or normal ("Normal").
func (c *Client) SetChatVisibility(ctx context.Context, accountID, chatID int, visibility string) error {
	_, err := c.caller.Call(ctx, "set_chat_visibility", []any{accountID, chatID, visibility})
	return err
}

// SetChatMuteDuration mutes ("Forever") or unmutes ("NotMuted") a chat.
func (c *Client) SetChatMuteDuration(ctx context.Context, accountID, chatID int, muted bool) error {
	dur := map[string]any{"kind": "NotMuted"}
	if muted {
		dur = map[string]any{"kind": "Forever"}
	}
	_, err := c.caller.Call(ctx, "set_chat_mute_duration", []any{accountID, chatID, dur})
	return err
}

// MaybeNetwork hints deltachat-rpc-server to immediately wake up IMAP IDLE & SMTP queues.
func (c *Client) MaybeNetwork(ctx context.Context) error {
	_, err := c.caller.Call(ctx, "maybe_network", []any{})
	return err
}

// GetContacts returns hydrated Delta Chat contacts for an account, optionally filtered by query.
func (c *Client) GetContacts(ctx context.Context, accountID int, query string) ([]Contact, error) {
	var q any
	if query != "" {
		q = query
	}
	raw, err := c.caller.Call(ctx, "get_contacts", []any{accountID, 0, q})
	if err != nil {
		return nil, err
	}
	var contacts []Contact
	if err := json.Unmarshal(raw, &contacts); err != nil {
		return nil, fmt.Errorf("dc: decode contacts: %w", err)
	}
	return contacts, nil
}

// GetChatContacts returns contact IDs belonging to a chat.
func (c *Client) GetChatContacts(ctx context.Context, accountID, chatID int) ([]int, error) {
	raw, err := c.caller.Call(ctx, "get_chat_contacts", []any{accountID, chatID})
	if err != nil {
		return nil, err
	}
	var ids []int
	if err := json.Unmarshal(raw, &ids); err != nil {
		return nil, fmt.Errorf("dc: decode chat contacts: %w", err)
	}
	return ids, nil
}
