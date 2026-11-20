package dc

import (
	"context"
	"encoding/json"
	"fmt"
)

// AddAccount creates a new unconfigured profile on deltachat-rpc-server.
func (c *Client) AddAccount(ctx context.Context) (int, error) {
	raw, err := c.transport.Call(ctx, "add_account", []any{})
	if err != nil {
		return 0, err
	}
	var id int
	if err := json.Unmarshal(raw, &id); err != nil {
		return 0, fmt.Errorf("dc: decode new account id: %w", err)
	}
	return id, nil
}

// GetConnectivity returns the numeric connectivity status (4000=connected, 3000=syncing, etc.).
func (c *Client) GetConnectivity(ctx context.Context, accountID int) (int, error) {
	raw, err := c.transport.Call(ctx, "get_connectivity", []any{accountID})
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
	data := map[string]any{
		"file": filePath,
	}
	if caption != "" {
		data["text"] = caption
	}
	raw, err := c.transport.Call(ctx, "send_msg", []any{accountID, chatID, data})
	if err != nil {
		return 0, err
	}
	var msgID int
	if err := json.Unmarshal(raw, &msgID); err != nil {
		return 0, fmt.Errorf("dc: decode sent file msg id: %w", err)
	}
	return msgID, nil
}

// DeleteMessages deletes one or more messages locally (Delete for Me).
func (c *Client) DeleteMessages(ctx context.Context, accountID int, msgIDs []int) error {
	if len(msgIDs) == 0 {
		return nil
	}
	_, err := c.transport.Call(ctx, "delete_messages", []any{accountID, msgIDs})
	return err
}

// SendEditRequest edits a previously sent outgoing message.
func (c *Client) SendEditRequest(ctx context.Context, accountID, msgID int, newText string) error {
	_, err := c.transport.Call(ctx, "send_edit_request", []any{accountID, msgID, newText})
	return err
}

// SetChatVisibility toggles pinned ("Pinned"), archived ("Archived"), or normal ("Normal").
func (c *Client) SetChatVisibility(ctx context.Context, accountID, chatID int, visibility string) error {
	_, err := c.transport.Call(ctx, "set_chat_visibility", []any{accountID, chatID, visibility})
	return err
}

// SetChatMuteDuration mutes ("Forever") or unmutes ("NotMuted") a chat.
func (c *Client) SetChatMuteDuration(ctx context.Context, accountID, chatID int, muted bool) error {
	dur := map[string]any{"kind": "NotMuted"}
	if muted {
		dur = map[string]any{"kind": "Forever"}
	}
	_, err := c.transport.Call(ctx, "set_chat_mute_duration", []any{accountID, chatID, dur})
	return err
}
