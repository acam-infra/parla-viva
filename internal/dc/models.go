package dc

import (
	"time"
	"unicode"
)

// ChatKind classifies a Delta Chat conversation (mirrors Parla's ChatKind).
type ChatKind int

const (
	ChatDirect ChatKind = iota
	ChatGroup
	ChatChannel
	ChatSavedMessages
	ChatDevice
)

// MessageState mirrors Delta Chat core message delivery states.
type MessageState int

const (
	MsgStateUndefined     MessageState = 0
	MsgStateInFresh       MessageState = 10
	MsgStateInNoticed     MessageState = 13
	MsgStateInSeen        MessageState = 16
	MsgStateOutPreparing  MessageState = 18
	MsgStateOutDraft      MessageState = 19
	MsgStateOutPending    MessageState = 20
	MsgStateOutFailed     MessageState = 24
	MsgStateOutDelivered  MessageState = 26
	MsgStateOutMdnRcvd    MessageState = 28
)

// Account represents a Delta Chat profile managed by deltachat-rpc-server.
type Account struct {
	ID          int    `json:"id"`
	Kind        string `json:"kind"` // "Configured" or "Unconfigured"
	DisplayName string `json:"displayName"`
	Addr        string `json:"addr"`
	Color       string `json:"color"`
	UnreadCount int    `json:"-"`
}

// Contact represents a Delta Chat address-book contact returned by get_contacts.
type Contact struct {
	ID              int    `json:"id"`
	Address         string `json:"address"`
	Name            string `json:"name"`
	DisplayName     string `json:"displayName"`
	AuthName        string `json:"authName"`
	Color           string `json:"color"`
	ProfileImage    string `json:"profileImage"`
	IsVerified      bool   `json:"isVerified"`
	IsBot           bool   `json:"isBot"`
	IsBlocked       bool   `json:"isBlocked"`
	WasSeenRecently bool   `json:"wasSeenRecently"`
	LastSeen        int64  `json:"lastSeen"`
	Status          string `json:"status"`
}

// DisplayLabel returns the preferred human-readable name for a contact.
func (c Contact) DisplayLabel() string {
	switch {
	case c.DisplayName != "":
		return c.DisplayName
	case c.Name != "":
		return c.Name
	case c.AuthName != "":
		return c.AuthName
	case c.Address != "":
		return c.Address
	default:
		return "Contact"
	}
}

// Initials returns up to two uppercase runes for avatar fallback.
func (c Contact) Initials() string {
	return initialsFromName(c.DisplayLabel())
}

// ChatEntry is a sidebar chatlist item parsed from get_chatlist_items_by_entries.
type ChatEntry struct {
	ID               int
	Name             string
	SummaryPrefix    string
	LastMessage      string
	LastMessageID    int
	UnreadCount      int
	Timestamp        int64 // Unix seconds
	AvatarPath       string
	Kind             ChatKind
	ChatType         string
	IsMuted          bool
	IsPinned         bool
	IsArchived       bool
	IsContactRequest bool
	IsDraft          bool
	WasSeenRecently  bool
}

// Initials returns up to two uppercase runes for avatar fallback.
func (c ChatEntry) Initials() string {
	return initialsFromName(c.Name)
}

// ShortTimeLabel formats the chat's last-activity timestamp like Telegram Desktop
// (HH:MM today, "Yesterday", weekday within 6 days, or "Jan 2" for older chats).
func (c ChatEntry) ShortTimeLabel() string {
	if c.Timestamp <= 0 {
		return ""
	}
	t := time.Unix(c.Timestamp, 0).Local()
	now := time.Now().Local()
	ty, tm, td := t.Date()
	ny, nm, nd := now.Date()
	if ty == ny && tm == nm && td == nd {
		return t.Format("15:04")
	}
	yest := now.AddDate(0, 0, -1)
	yy, ym, yd := yest.Date()
	if ty == yy && tm == ym && td == yd {
		return "Yesterday"
	}
	if now.Sub(t) > 0 && now.Sub(t) < 6*24*time.Hour {
		return t.Format("Mon")
	}
	return t.Format("Jan 2")
}

// Reaction aggregates an emoji reaction on a message.
type Reaction struct {
	Emoji   string
	Count   int
	BySelf  bool
}

// Quote represents a replied-to message preview.
type Quote struct {
	MessageID  int
	AuthorName string
	Text       string
}

// Message represents a single Delta Chat message (mirrors Parla's Message model).
type Message struct {
	ID                 int
	ChatID             int
	Text               string
	Timestamp          int64 // Unix seconds
	IsInfo             bool
	IsForwarded        bool
	IsEdited           bool
	IsPinned           bool
	IsOutgoing         bool
	SenderContactID    int
	SenderName         string
	SenderAddress      string
	OverrideSenderName string
	FilePath           string
	FileName           string
	FileMime           string
	FileBytes          int64
	ViewType           string // "Text", "Image", "Gif", "Sticker", "Audio", "Voice", "Video", "File", "Webxdc", "Vcard"
	State              MessageState
	Error              string
	HasHTML            bool
	DownloadState      string
	Reactions          []Reaction
	Quote              *Quote
}

// DisplaySender returns the effective display name for the message author.
func (m Message) DisplaySender() string {
	if m.OverrideSenderName != "" {
		return m.OverrideSenderName
	}
	if m.SenderName != "" {
		return m.SenderName
	}
	if m.SenderAddress != "" {
		return m.SenderAddress
	}
	if m.IsOutgoing {
		return "You"
	}
	return "Unknown"
}

// TimeLabel formats the message timestamp as HH:MM.
func (m Message) TimeLabel() string {
	if m.Timestamp <= 0 {
		return ""
	}
	return time.Unix(m.Timestamp, 0).Format("15:04")
}

// StatusLabel returns a short delivery state badge for outgoing messages.
func (m Message) StatusLabel() string {
	if !m.IsOutgoing {
		return ""
	}
	switch m.State {
	case MsgStateOutPending, MsgStateOutPreparing:
		return "Sending"
	case MsgStateOutFailed:
		return "Failed"
	case MsgStateOutDelivered:
		return "Delivered"
	case MsgStateOutMdnRcvd:
		return "Read"
	default:
		return ""
	}
}

// SearchResult represents a cross-chat or in-chat message search hit.
type SearchResult struct {
	MessageID  int
	ChatID     int
	ChatName   string
	AuthorName string
	Text       string
	Timestamp  int64
	IsArchived bool
}

// Event is a decoded event envelope from get_next_event.
type Event struct {
	ContextID int            `json:"contextId"`
	Event     EventPayload   `json:"event"`
}

// EventPayload carries the discriminated Delta Chat core event kind and fields.
type EventPayload struct {
	Kind      string `json:"kind"`
	ChatID    int    `json:"chatId,omitempty"`
	MsgID     int    `json:"msgId,omitempty"`
	ContactID int    `json:"contactId,omitempty"`
	Progress  int    `json:"progress,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Reaction  string `json:"reaction,omitempty"`
}

// DateLabel formats the message calendar day (Today / Yesterday / YYYY-MM-DD).
func (m Message) DateLabel() string {
	if m.Timestamp <= 0 {
		return ""
	}
	t := time.Unix(m.Timestamp, 0)
	now := time.Now()
	y1, m1, d1 := t.Date()
	y2, m2, d2 := now.Date()
	if y1 == y2 && m1 == m2 && d1 == d2 {
		return "Today"
	}
	yest := now.AddDate(0, 0, -1)
	y3, m3, d3 := yest.Date()
	if y1 == y3 && m1 == m3 && d1 == d3 {
		return "Yesterday"
	}
	return t.Format("2006-01-02")
}

func initialsFromName(name string) string {
	var out []rune
	takeNext := true
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			takeNext = true
			continue
		}
		if takeNext {
			out = append(out, unicode.ToUpper(r))
			if len(out) == 2 {
				break
			}
			takeNext = false
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return string(out)
}
