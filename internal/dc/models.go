package dc

import "time"

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

func initialsFromName(name string) string {
	runes := []rune(name)
	var out []rune
	takeNext := true
	for _, r := range runes {
		if r == ' ' || r == '-' || r == '_' || r == '.' {
			takeNext = true
			continue
		}
		if takeNext {
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A'
			}
			out = append(out, r)
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
