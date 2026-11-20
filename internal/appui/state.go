package appui

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/history"
)

// LayoutMode selects one of Parla's three conversation layouts.
type LayoutMode string

const (
	LayoutBubble    LayoutMode = "Bubble"
	LayoutCompact   LayoutMode = "Compact IRC"
	LayoutWorkspace LayoutMode = "Workspace"
)

var LayoutModes = []LayoutMode{LayoutBubble, LayoutCompact, LayoutWorkspace}

var ChatFilters = []string{"All", "Unread", "Pinned", "Groups"}

var GalleryTabs = []string{"All", "Images", "Files", "Webxdc"}

// Model holds all application state for Parla Viva.
type Model struct {
	mu sync.Mutex

	// Live RPC client (nil when running in offline demo mode).
	Client       *dc.Client
	Invalidator  *app.Invalidator
	WebHost      *app.WebHost
	WebView      *app.WebView
	WebReady     bool
	AccountsPath string
	DemoMode     bool
	StatusText   string

	// Accounts & active chat state
	Accounts     []dc.Account
	ActiveAcctID int
	Chats        []dc.ChatEntry
	ActiveChatID int
	Messages     map[int][]dc.Message
	Windows      map[int]history.Window
	NextLocalID  int

	// Image bitmap cache & Lightbox viewer
	ImageCache     map[string]gfx.Bitmap
	LightboxOpen   bool
	LightboxTitle  string
	LightboxBitmap gfx.Bitmap
	LightboxZoom   float32

	// Navigation & search
	SidebarFilter string
	SidebarQuery  string
	SearchOpen    bool
	SearchQuery   string
	SearchMatch   int
	QuickOpen     bool
	QuickQuery    string

	// Composer, multi-image album queue & drag-and-drop
	Draft        string
	PendingFiles []PendingAttachment
	DragOver     bool
	ReplyTo      *dc.Message
	EditingMsgID int
	Selecting    bool
	SelectedMsgs map[int]bool
	NewChatOpen  bool
	NewChatName  string
	NewChatAddr  string

	// Chatmail Onboarding / Invite & Join modal
	InviteOpen    bool
	InviteLink    string
	JoinLinkInput string
	NewAcctName   string
	NewAcctRelay  string
	OnboardStatus string

	// Dialogs & drawers
	AccountsOpen   bool
	ChatInfoOpen   bool
	GalleryOpen    bool
	GalleryTab     string
	SettingsOpen   bool
	BackupPassword string
	LastBackupPath string
	BackupStatus   string

	// Appearance & preferences
	DarkMode      bool
	Theme         gfx.Theme
	Layout        LayoutMode
	EnterToSend   bool
	RenderMD      bool
	NotifyEnabled bool
	Split         ui.SplitterState

	// Webxdc drawer preview
	WebxdcOpen  bool
	WebxdcTitle string
	WebxdcInfo  string

	// Notifications
	Toaster ui.Toaster

	// Pending async updates drained in OnWake
	pendingUpdates []func(*Model)
}

// NewDemoModel constructs an interactive offline preview populated with realistic
// encrypted Delta Chat conversations and sample inline images.
func NewDemoModel() *Model {
	now := time.Now().Unix()
	m := &Model{
		DemoMode:      true,
		StatusText:    "Demo mode (deltachat-rpc-server not connected)",
		ActiveAcctID:  1,
		ActiveChatID:  10,
		SidebarFilter: "All",
		GalleryTab:    "All",
		Layout:        LayoutBubble,
		EnterToSend:   true,
		RenderMD:      true,
		NotifyEnabled: true,
		Theme:         gfx.LightTheme(),
		Split:         ui.SplitterState{Pos: 320},
		Messages:      make(map[int][]dc.Message),
		Windows:       make(map[int]history.Window),
		SelectedMsgs:  make(map[int]bool),
		ImageCache:    make(map[string]gfx.Bitmap),
		NewAcctRelay:  dc.DefaultChatmailRelay,
		InviteLink:    "https://i.delta.chat/#demo-invite-preview",
		NextLocalID:   5000,
		Accounts: []dc.Account{
			{ID: 1, Kind: "Configured", DisplayName: "Alice", Addr: "alice@nine.testrun.org"},
			{ID: 2, Kind: "Configured", DisplayName: "Work Profile", Addr: "dev@chatmail.example"},
		},
		Chats: []dc.ChatEntry{
			{
				ID: 10, Name: "Parla → Viva Migration", Kind: dc.ChatGroup, ChatType: "Group",
				SummaryPrefix: "Bob", LastMessage: "Shared desktop screenshot & multi-image support!",
				UnreadCount: 2, Timestamp: now - 120, IsPinned: true, WasSeenRecently: true,
			},
			{
				ID: 11, Name: "Elena (Chatmail)", Kind: dc.ChatDirect, ChatType: "Single",
				LastMessage: "Shared Webxdc poll: Release checklist",
				UnreadCount: 1, Timestamp: now - 900, WasSeenRecently: true,
			},
			{
				ID: 12, Name: "Saved Messages", Kind: dc.ChatSavedMessages, ChatType: "Single",
				LastMessage: "viva package --zip builds single native binary",
				UnreadCount: 0, Timestamp: now - 3600, IsPinned: true,
			},
			{
				ID: 13, Name: "Linux Desktop Channel", Kind: dc.ChatChannel, ChatType: "Broadcast",
				LastMessage: "Vulkan + CPU fallback verified on Debian X11",
				UnreadCount: 0, Timestamp: now - 7200, IsMuted: true,
			},
		},
	}

	// Pre-populate a crisp sample bitmap for the demo image message.
	m.ImageCache["demo://preview-desktop.png"] = DemoGradientBitmap(320, 200, 3)

	m.Messages[10] = []dc.Message{
		{
			ID: 1001, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Welcome to Parla Viva! Right-click any chat or message for Telegram-style quick actions.",
			Timestamp: now - 1800, ViewType: "Text", State: dc.MsgStateInSeen, IsPinned: true,
			Reactions: []dc.Reaction{{Emoji: "👍", Count: 2, BySelf: true}},
		},
		{
			ID: 1002, ChatID: 10, SenderContactID: 1, SenderName: "Alice", IsOutgoing: true,
			Text:      "No GTK4/libadwaita/MSYS2 bundling needed — Viva renders directly with Metal, Direct3D 11, or Vulkan.",
			Timestamp: now - 1200, ViewType: "Text", State: dc.MsgStateOutMdnRcvd,
		},
		{
			ID: 1003, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Click the image below to open the Lightbox zoom viewer, or drag & drop multiple photos into the window.",
			Timestamp: now - 120, ViewType: "Image", FilePath: "demo://preview-desktop.png", FileName: "preview-desktop.png",
			FileBytes: 184320, State: dc.MsgStateInFresh,
			Quote:     &dc.Quote{MessageID: 1002, AuthorName: "Alice", Text: "No GTK4/libadwaita/MSYS2 bundling needed..."},
			Reactions: []dc.Reaction{{Emoji: "🔥", Count: 3}},
		},
		{
			ID: 1004, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Attached architecture spec for review.",
			Timestamp: now - 60, ViewType: "File", FileName: "architecture-v0.2.pdf", FileBytes: 248320, State: dc.MsgStateInFresh,
		},
	}

	m.Messages[11] = []dc.Message{
		{
			ID: 2001, ChatID: 11, SenderContactID: 3, SenderName: "Elena",
			Text:      "Hey! I sent a Webxdc release checklist mini-app. Click Open Webxdc to inspect the offline sandbox bridge.",
			Timestamp: now - 900, ViewType: "Webxdc", FileName: "checklist.xdc", State: dc.MsgStateInFresh,
		},
	}

	m.Messages[12] = []dc.Message{
		{
			ID: 3001, ChatID: 12, SenderContactID: 1, SenderName: "Alice", IsOutgoing: true,
			Text:      "viva package --zip builds a single native binary with Ed25519 signed updates.",
			Timestamp: now - 3600, ViewType: "Text", State: dc.MsgStateOutDelivered,
		},
	}

	m.Messages[13] = []dc.Message{
		{
			ID: 4001, ChatID: 13, SenderContactID: 4, SenderName: "Maintainer",
			Text:      "Vulkan + CPU fallback verified on Debian X11 and Windows 11 D3D11.",
			Timestamp: now - 7200, ViewType: "Text", State: dc.MsgStateInSeen,
		},
	}

	for chatID, msgs := range m.Messages {
		ids := make([]int, len(msgs))
		for i, msg := range msgs {
			ids[i] = msg.ID
		}
		m.Windows[chatID] = history.NewLatest(ids, 0, history.DefaultBatchSize)
	}
	return m
}

// NewLiveModel initializes a Model backed by a connected dc.Client.
func NewLiveModel(client *dc.Client, invalidator *app.Invalidator) *Model {
	m := NewDemoModel()
	m.DemoMode = false
	m.Client = client
	m.Invalidator = invalidator
	m.StatusText = "Connected to deltachat-rpc-server"
	m.Chats = nil
	m.Messages = make(map[int][]dc.Message)
	m.Windows = make(map[int]history.Window)
	m.ActiveChatID = 0
	m.InviteLink = ""
	return m
}

// EnqueueUpdate schedules a state mutation to run safely on the window thread via OnWake.
func (m *Model) EnqueueUpdate(fn func(*Model)) {
	m.mu.Lock()
	m.pendingUpdates = append(m.pendingUpdates, fn)
	inv := m.Invalidator
	m.mu.Unlock()
	if inv != nil {
		inv.Invalidate()
	}
}

// DrainUpdates applies all queued background updates on the window thread.
func (m *Model) DrainUpdates() {
	m.mu.Lock()
	updates := m.pendingUpdates
	m.pendingUpdates = nil
	m.mu.Unlock()
	for _, fn := range updates {
		fn(m)
	}
}

// ActiveAccount returns the currently selected Account or nil.
func (m *Model) ActiveAccount() *dc.Account {
	for i := range m.Accounts {
		if m.Accounts[i].ID == m.ActiveAcctID {
			return &m.Accounts[i]
		}
	}
	if len(m.Accounts) > 0 {
		return &m.Accounts[0]
	}
	return nil
}

// FilteredChats returns chats matching the current sidebar filter and search query.
func (m *Model) FilteredChats() []dc.ChatEntry {
	q := strings.ToLower(strings.TrimSpace(m.SidebarQuery))
	out := make([]dc.ChatEntry, 0, len(m.Chats))
	for _, ch := range m.Chats {
		switch m.SidebarFilter {
		case "Unread":
			if ch.UnreadCount == 0 {
				continue
			}
		case "Pinned":
			if !ch.IsPinned {
				continue
			}
		case "Groups":
			if ch.Kind != dc.ChatGroup && ch.Kind != dc.ChatChannel {
				continue
			}
		}
		if q != "" && !strings.Contains(strings.ToLower(ch.Name), q) &&
			!strings.Contains(strings.ToLower(ch.LastMessage), q) {
			continue
		}
		out = append(out, ch)
	}
	return out
}

// ActiveChat returns a pointer to the currently active chat entry, or nil.
func (m *Model) ActiveChat() *dc.ChatEntry {
	for i := range m.Chats {
		if m.Chats[i].ID == m.ActiveChatID {
			return &m.Chats[i]
		}
	}
	return nil
}

// SelectChat switches the active conversation and clears its unread badge.
func (m *Model) SelectChat(chatID int) {
	m.ActiveChatID = chatID
	m.ReplyTo = nil
	m.EditingMsgID = 0
	m.Selecting = false
	m.SelectedMsgs = make(map[int]bool)
	for i := range m.Chats {
		if m.Chats[i].ID == chatID {
			m.Chats[i].UnreadCount = 0
			break
		}
	}
	if !m.DemoMode && m.Client != nil {
		acctID := m.ActiveAcctID
		go m.loadChatMessagesAsync(acctID, chatID)
	}
}

func (m *Model) loadChatMessagesAsync(acctID, chatID int) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ids, err := m.Client.GetMessageIDs(ctx, acctID, chatID)
	if err != nil {
		return
	}
	win := history.NewLatest(ids, 0, history.DefaultBatchSize)
	msgs, err := m.Client.GetMessages(ctx, acctID, win.VisibleIDs())
	if err != nil {
		return
	}
	_ = m.Client.MarkSeenMsgs(ctx, acctID, win.VisibleIDs())
	m.EnqueueUpdate(func(model *Model) {
		model.Windows[chatID] = win
		model.Messages[chatID] = msgs
	})
}

// SendDraft sends or edits the current composer text and any queued multi-image/file attachments.
func (m *Model) SendDraft() {
	text := strings.TrimSpace(m.Draft)
	files := append([]PendingAttachment(nil), m.PendingFiles...)
	if (text == "" && len(files) == 0) || m.ActiveChatID == 0 {
		return
	}
	chatID := m.ActiveChatID

	if m.EditingMsgID > 0 {
		msgID := m.EditingMsgID
		m.EditingMsgID = 0
		m.Draft = ""
		m.ApplyEditMessage(chatID, msgID, text)
		return
	}

	var quote *dc.Quote
	quotedID := 0
	if m.ReplyTo != nil {
		quotedID = m.ReplyTo.ID
		quote = &dc.Quote{
			MessageID:  m.ReplyTo.ID,
			AuthorName: m.ReplyTo.DisplaySender(),
			Text:       m.ReplyTo.Text,
		}
	}
	m.Draft = ""
	m.PendingFiles = nil
	m.ReplyTo = nil

	if m.DemoMode || m.Client == nil {
		now := time.Now().Unix()
		if len(files) == 0 {
			m.NextLocalID++
			msg := dc.Message{
				ID: m.NextLocalID, ChatID: chatID, SenderContactID: 1, SenderName: "Alice",
				IsOutgoing: true, Text: text, Timestamp: now, ViewType: "Text",
				State: dc.MsgStateOutDelivered, Quote: quote,
			}
			m.Messages[chatID] = append(m.Messages[chatID], msg)
			m.updateChatPreview(chatID, text, now)
			return
		}
		for idx, att := range files {
			m.NextLocalID++
			vtype := "File"
			if att.IsImage {
				vtype = "Image"
				if att.Bitmap.Width > 0 {
					m.ImageCache[att.Path] = att.Bitmap
				}
			}
			caption := ""
			if idx == 0 {
				caption = text
			}
			var q *dc.Quote
			if idx == 0 {
				q = quote
			}
			msg := dc.Message{
				ID: m.NextLocalID, ChatID: chatID, SenderContactID: 1, SenderName: "Alice",
				IsOutgoing: true, Text: caption, FilePath: att.Path, FileName: att.Name,
				Timestamp: now, ViewType: vtype, State: dc.MsgStateOutDelivered, Quote: q,
			}
			m.Messages[chatID] = append(m.Messages[chatID], msg)
		}
		preview := fmt.Sprintf("Sent %d attachments", len(files))
		if len(files) == 1 {
			preview = "Attachment: " + files[0].Name
		}
		if text != "" {
			preview = text
		}
		m.updateChatPreview(chatID, preview, now)
		return
	}

	acctID := m.ActiveAcctID
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if len(files) == 0 {
			if _, err := m.Client.SendTextMessage(ctx, acctID, chatID, text, quotedID); err != nil {
				m.EnqueueUpdate(func(model *Model) {
					model.Toaster.Show("Send failed", err.Error(), ui.ToastError)
				})
				return
			}
		} else {
			for idx, att := range files {
				caption := ""
				if idx == 0 {
					caption = text
				}
				if _, err := m.Client.SendFileMessage(ctx, acctID, chatID, att.Path, caption); err != nil {
					m.EnqueueUpdate(func(model *Model) {
						model.Toaster.Show("Upload failed", att.Name+": "+err.Error(), ui.ToastError)
					})
					break
				}
			}
		}
		m.loadChatMessagesAsync(acctID, chatID)
	}()
}

func (m *Model) updateChatPreview(chatID int, preview string, ts int64) {
	for i := range m.Chats {
		if m.Chats[i].ID == chatID {
			m.Chats[i].LastMessage = preview
			m.Chats[i].SummaryPrefix = "You"
			m.Chats[i].Timestamp = ts
		}
	}
}

// ToggleReaction toggles an emoji reaction on a message.
func (m *Model) ToggleReaction(chatID, msgID int, emoji string) {
	msgs := m.Messages[chatID]
	for i := range msgs {
		if msgs[i].ID != msgID {
			continue
		}
		found := false
		for j := range msgs[i].Reactions {
			if msgs[i].Reactions[j].Emoji == emoji {
				found = true
				if msgs[i].Reactions[j].BySelf {
					msgs[i].Reactions[j].BySelf = false
					msgs[i].Reactions[j].Count--
					if msgs[i].Reactions[j].Count <= 0 {
						msgs[i].Reactions = append(msgs[i].Reactions[:j], msgs[i].Reactions[j+1:]...)
					}
				} else {
					msgs[i].Reactions[j].BySelf = true
					msgs[i].Reactions[j].Count++
				}
				break
			}
		}
		if !found {
			msgs[i].Reactions = append(msgs[i].Reactions, dc.Reaction{Emoji: emoji, Count: 1, BySelf: true})
		}
		break
	}
	m.Messages[chatID] = msgs

	if !m.DemoMode && m.Client != nil {
		acctID := m.ActiveAcctID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.Client.SendReaction(ctx, acctID, msgID, emoji)
		}()
	}
}

// CreateDemoChat creates a new local chat entry (used in demo mode or quick onboarding).
func (m *Model) CreateDemoChat(name, addr string) {
	name = strings.TrimSpace(name)
	if name == "" {
		return
	}
	m.NextLocalID++
	chatID := m.NextLocalID
	m.Chats = append([]dc.ChatEntry{{
		ID:          chatID,
		Name:        name,
		Kind:        dc.ChatDirect,
		ChatType:    "Single",
		LastMessage: fmt.Sprintf("Encrypted chat started with %s", addr),
		Timestamp:   time.Now().Unix(),
	}}, m.Chats...)
	m.Messages[chatID] = []dc.Message{{
		ID:        chatID*10 + 1,
		ChatID:    chatID,
		IsInfo:    true,
		Text:      "End-to-end encryption enabled via Autocrypt / Chatmail.",
		Timestamp: time.Now().Unix(),
	}}
	m.ActiveChatID = chatID
	m.NewChatOpen = false
	m.NewChatName = ""
	m.NewChatAddr = ""
	m.Toaster.Show("Chat created", name, ui.ToastSuccess)
}

// SetDarkMode switches between Viva's built-in LightTheme and DarkTheme.
func (m *Model) SetDarkMode(dark bool) {
	m.DarkMode = dark
	if dark {
		m.Theme = gfx.DarkTheme()
	} else {
		m.Theme = gfx.LightTheme()
	}
}

// PinnedMessages returns all pinned messages in the active conversation.
func (m *Model) PinnedMessages() []dc.Message {
	var out []dc.Message
	for _, msg := range m.Messages[m.ActiveChatID] {
		if msg.IsPinned {
			out = append(out, msg)
		}
	}
	return out
}

// SearchMatches returns indices of messages in the active chat matching SearchQuery.
func (m *Model) SearchMatches() []int {
	q := strings.ToLower(strings.TrimSpace(m.SearchQuery))
	if q == "" {
		return nil
	}
	var out []int
	for i, msg := range m.Messages[m.ActiveChatID] {
		if strings.Contains(strings.ToLower(msg.Text), q) ||
			strings.Contains(strings.ToLower(msg.DisplaySender()), q) {
			out = append(out, i)
		}
	}
	return out
}

// StartLiveSync loads initial accounts & chatlist and starts the event loop.
func (m *Model) StartLiveSync(ctx context.Context) {
	if m.Client == nil {
		return
	}
	go func() {
		accounts, err := m.Client.GetAllAccounts(ctx)
		if err != nil || len(accounts) == 0 {
			return
		}
		acctID := accounts[0].ID
		_ = m.Client.SelectAccount(ctx, acctID)
		_ = m.Client.StartIOForAllAccounts(ctx)
		ids, _ := m.Client.GetChatlistEntries(ctx, acctID, nil, "")
		chats, _ := m.Client.GetChatlistItems(ctx, acctID, ids)
		qr, _ := m.Client.GetChatSecurejoinQRCode(ctx, acctID, 0)

		m.EnqueueUpdate(func(model *Model) {
			model.Accounts = accounts
			model.ActiveAcctID = acctID
			model.Chats = chats
			model.InviteLink = qr
			if len(chats) > 0 && model.ActiveChatID == 0 {
				model.SelectChat(chats[0].ID)
			}
		})

		for {
			ev, err := m.Client.GetNextEvent(ctx)
			if err != nil {
				return
			}
			switch ev.Event.Kind {
			case "IncomingMsg", "MsgsChanged", "MsgDelivered", "MsgRead", "ReactionsChanged":
				chatIDs, err := m.Client.GetChatlistEntries(ctx, acctID, nil, "")
				if err == nil {
					updatedChats, _ := m.Client.GetChatlistItems(ctx, acctID, chatIDs)
					m.EnqueueUpdate(func(model *Model) {
						model.Chats = updatedChats
						if ev.Event.ChatID == model.ActiveChatID && ev.Event.ChatID > 0 {
							go model.loadChatMessagesAsync(acctID, ev.Event.ChatID)
						}
					})
				}
			}
		}
	}()
}

// MessageBitmap returns a cached gfx.Bitmap for an image message, decoding lazily if needed.
func (m *Model) MessageBitmap(msg dc.Message) (gfx.Bitmap, bool) {
	if msg.FilePath == "" {
		return gfx.Bitmap{}, false
	}
	if bmp, ok := m.ImageCache[msg.FilePath]; ok && bmp.Width > 0 {
		return bmp, true
	}
	if !IsImagePath(msg.FilePath) && msg.ViewType != "Image" {
		return gfx.Bitmap{}, false
	}
	if bmp, ok := LoadBitmap(msg.FilePath, 480); ok {
		m.ImageCache[msg.FilePath] = bmp
		return bmp, true
	}
	return gfx.Bitmap{}, false
}

// AddAttachments queues one or more files/images for multi-attachment upload.
func (m *Model) AddAttachments(paths []string) {
	for i, p := range paths {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		name := filepath.Base(p)
		isImg := IsImagePath(p)
		var bmp gfx.Bitmap
		if isImg {
			if loaded, ok := LoadBitmap(p, 240); ok {
				bmp = loaded
			} else {
				bmp = DemoGradientBitmap(160, 120, byte(len(m.PendingFiles)+i+1))
			}
		}
		m.PendingFiles = append(m.PendingFiles, PendingAttachment{
			Path:    p,
			Name:    name,
			IsImage: isImg,
			Bitmap:  bmp,
		})
	}
}

// RemoveAttachment removes one queued attachment by index.
func (m *Model) RemoveAttachment(idx int) {
	if idx < 0 || idx >= len(m.PendingFiles) {
		return
	}
	m.PendingFiles = append(m.PendingFiles[:idx], m.PendingFiles[idx+1:]...)
}

// HandleFileDrop implements app.Config.OnFileDrop for Telegram-style drag-and-drop multi-upload.
func (m *Model) HandleFileDrop(ev gfx.FileDropEvent) gfx.EventResult {
	switch ev.Phase {
	case gfx.FileDropOver:
		m.DragOver = true
		return gfx.EventResult{Handled: true, Repaint: true}
	case gfx.FileDropLeave:
		m.DragOver = false
		return gfx.EventResult{Handled: true, Repaint: true}
	case gfx.FileDropDrop:
		m.DragOver = false
		if len(ev.Paths) > 0 {
			m.AddAttachments(ev.Paths)
			m.Toaster.Show("Files attached", fmt.Sprintf("%d item(s) ready to send", len(ev.Paths)), ui.ToastInfo)
		}
		return gfx.EventResult{Handled: true, Repaint: true}
	}
	return gfx.EventResult{}
}
