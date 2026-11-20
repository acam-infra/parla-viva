package dc

import (
	"context"
	"encoding/json"
	"fmt"
)

// DefaultChatmailRelay is the default public Chatmail relay used for 1-click onboarding.
const DefaultChatmailRelay = "dcaccount:nine.testrun.org"

// SetConfig sets a configuration key (such as "displayname") for an account.
func (c *Client) SetConfig(ctx context.Context, accountID int, key, value string) error {
	_, err := c.transport.Call(ctx, "set_config", []any{accountID, key, value})
	return err
}

// GetConfig reads a string configuration key for an account.
func (c *Client) GetConfig(ctx context.Context, accountID int, key string) (string, error) {
	raw, err := c.transport.Call(ctx, "get_config", []any{accountID, key})
	if err != nil {
		return "", err
	}
	if string(raw) == "null" {
		return "", nil
	}
	var val string
	if err := json.Unmarshal(raw, &val); err != nil {
		return "", fmt.Errorf("dc: decode config %s: %w", key, err)
	}
	return val, nil
}

// AddTransportFromQR configures a Chatmail or email transport from a dcaccount:/dclogin: URI.
func (c *Client) AddTransportFromQR(ctx context.Context, accountID int, qrURI string) error {
	_, err := c.transport.Call(ctx, "add_transport_from_qr", []any{accountID, qrURI})
	return err
}

// GetChatSecurejoinQRCode returns the shareable invite URL (https://i.delta.chat/#...)
// for either the account (chatID == 0) or a specific group (chatID > 0).
func (c *Client) GetChatSecurejoinQRCode(ctx context.Context, accountID int, chatID int) (string, error) {
	var cid any
	if chatID > 0 {
		cid = chatID
	}
	raw, err := c.transport.Call(ctx, "get_chat_securejoin_qr_code", []any{accountID, cid})
	if err != nil {
		return "", err
	}
	var qr string
	if err := json.Unmarshal(raw, &qr); err != nil {
		return "", fmt.Errorf("dc: decode invite qr: %w", err)
	}
	return qr, nil
}

// SecureJoin redeems a peer's invite link (https://i.delta.chat/#... or openpgp4fpr:...)
// and returns the newly opened encrypted chat ID.
func (c *Client) SecureJoin(ctx context.Context, accountID int, qrURI string) (int, error) {
	raw, err := c.transport.Call(ctx, "secure_join", []any{accountID, qrURI})
	if err != nil {
		return 0, err
	}
	var chatID int
	if err := json.Unmarshal(raw, &chatID); err != nil {
		return 0, fmt.Errorf("dc: decode secure_join chat id: %w", err)
	}
	return chatID, nil
}

// CreateChatmailAccount creates a new account, sets displayName, provisions it on relayURI
// (defaults to nine.testrun.org), starts I/O, and returns the account ID and invite link.
func (c *Client) CreateChatmailAccount(ctx context.Context, displayName, relayURI string) (int, string, error) {
	if relayURI == "" {
		relayURI = DefaultChatmailRelay
	}
	id, err := c.AddAccount(ctx)
	if err != nil {
		return 0, "", err
	}
	if displayName != "" {
		if err := c.SetConfig(ctx, id, "displayname", displayName); err != nil {
			return id, "", err
		}
	}
	if err := c.AddTransportFromQR(ctx, id, relayURI); err != nil {
		return id, "", err
	}
	_ = c.SelectAccount(ctx, id)
	_ = c.StartIOForAllAccounts(ctx)
	qr, err := c.GetChatSecurejoinQRCode(ctx, id, 0)
	return id, qr, err
}
