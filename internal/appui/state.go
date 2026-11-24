package appui

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"
	uimarkdown "github.com/vivaui/viva/ui/markdown"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/history"
	"github.com/acam-infra/parla-viva/internal/notify"
	"github.com/acam-infra/parla-viva/internal/preview"
)

// LayoutMode selects one of Parla's three conversation layouts.
type LayoutMode string

const (
	LayoutBubble    LayoutMode = "Bubble"
	LayoutCompact   LayoutMode = "Compact IRC"
	LayoutWorkspace LayoutMode = "Workspace"
)

var LayoutModes = []LayoutMode{LayoutBubble, LayoutCompact, LayoutWorkspace}

var ChatFilters = []string{"All", "Unread", "Pinned"}

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
	OnEvent      func(dc.Event)
	OnQuit       func()
	Notifier     *notify.Notifier
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

	// Image bitmap cache & Universal File Previewer state
	ImageCache    map[string]gfx.Bitmap
	PreviewOpen   bool
	PreviewDoc    preview.Document
	PreviewBitmap gfx.Bitmap
	PreviewZoom   float32
	PreviewMD     uimarkdown.State
	PreviewTable  ui.TableState
	PreviewRows   ui.TableRows

	// Navigation, header menus & search
	SidebarFilter string
	SidebarQuery  string
	SearchOpen    bool
	SearchQuery   string
	SearchMatch   int
	QuickOpen     bool
	QuickQuery    string
	NewMenuOpen   bool
	MoreMenuOpen  bool

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

func newBaseModel() *Model {
	return &Model{
		SidebarFilter: "All",
		GalleryTab:    "All",
		Layout:        LayoutBubble,
		EnterToSend:   true,
		RenderMD:      true,
		NotifyEnabled: true,
		Theme:         gfx.LightTheme(),
		Split:         ui.SplitterState{Pos: 300},
		Messages:      make(map[int][]dc.Message),
		Windows:       make(map[int]history.Window),
		SelectedMsgs:  make(map[int]bool),
		ImageCache:    make(map[string]gfx.Bitmap),
		NewAcctRelay:  dc.DefaultChatmailRelay,
		NextLocalID:   5000,
	}
}

// NewDemoModel constructs an interactive offline preview populated with realistic
// encrypted Delta Chat conversations and common-format attachment fixtures.
func NewDemoModel() *Model {
	now := time.Now().Unix()
	m := newBaseModel()
	m.DemoMode = true
	m.StatusText = "Offline Preview (start with deltachat-rpc-server for live chat)"
	m.ActiveAcctID = 1
	m.ActiveChatID = 10
	m.InviteLink = "https://i.delta.chat/#demo-invite-preview"
	m.Accounts = []dc.Account{
		{ID: 1, Kind: "Configured", DisplayName: "Alice", Addr: "alice@nine.testrun.org"},
		{ID: 2, Kind: "Configured", DisplayName: "Work Profile", Addr: "dev@chatmail.example"},
	}
	m.Chats = []dc.ChatEntry{
		{
			ID: 10, Name: "Parla → Viva Migration", Kind: dc.ChatGroup, ChatType: "Group",
			SummaryPrefix: "Bob", LastMessage: "Shared desktop preview, notes.md & metrics.csv",
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
	}

	m.ImageCache["demo://preview-desktop.png"] = DemoGradientBitmap(320, 200, 3)

	m.Messages[10] = []dc.Message{
		{
			ID: 1001, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Welcome to Parla Viva! Right-click any chat or message for Telegram-style actions.",
			Timestamp: now - 86400, ViewType: "Text", State: dc.MsgStateInSeen, IsPinned: true,
			Reactions: []dc.Reaction{{Emoji: "👍", Count: 2, BySelf: true}},
		},
		{
			ID: 1002, ChatID: 10, SenderContactID: 1, SenderName: "Alice", IsOutgoing: true,
			Text:      "Unified Telegram-style UI with direct in-app previews for images, Markdown, JSON, CSV tables, code, ZIP & vCard.",
			Timestamp: now - 1200, ViewType: "Text", State: dc.MsgStateOutMdnRcvd,
		},
		{
			ID: 1003, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Click the image to zoom in, or drag & drop multiple photos directly into the window.",
			Timestamp: now - 300, ViewType: "Image", FilePath: "demo://preview-desktop.png", FileName: "preview-desktop.png",
			FileBytes: 184320, State: dc.MsgStateInFresh,
			Quote:     &dc.Quote{MessageID: 1002, AuthorName: "Alice", Text: "Unified Telegram-style UI with direct in-app previews..."},
			Reactions: []dc.Reaction{{Emoji: "🔥", Count: 3}},
		},
		{
			ID: 1004, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Release notes (click to preview rendered Markdown inside the app):",
			Timestamp: now - 120, ViewType: "File", FilePath: "demo://release-notes.md", FileName: "release-notes.md",
			FileBytes: 1024, State: dc.MsgStateInFresh,
		},
		{
			ID: 1005, ChatID: 10, SenderContactID: 2, SenderName: "Bob",
			Text:      "Cross-platform build metrics (click to preview as interactive Table):",
			Timestamp: now - 60, ViewType: "File", FilePath: "demo://build-metrics.csv", FileName: "build-metrics.csv",
			FileBytes: 512, State: dc.MsgStateInFresh,
		},
	}

	m.Messages[11] = []dc.Message{
		{
			ID: 2001, ChatID: 11, SenderContactID: 3, SenderName: "Elena",
			Text:      "Hey! I sent a Webxdc release checklist mini-app.",
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

// NewLiveModel initializes a clean Model backed by a real deltachat-rpc-server
// with zero fake demo accounts or placeholder chats.
func NewLiveModel(client *dc.Client, invalidator *app.Invalidator) *Model {
	m := newBaseModel()
	m.DemoMode = false
	m.Client = client
	m.Invalidator = invalidator
	m.Notifier = notify.New()
	m.StatusText = "Connected to deltachat-rpc-server"
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

// currentAccountID safely reads ActiveAcctID for background sync goroutines.
func (m *Model) currentAccountID() int {
	m.mu.Lock()
	id := m.ActiveAcctID
	m.mu.Unlock()
	return id
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
	m.deltaSyncChatAsync(acctID, chatID, true)
}

func (m *Model) deltaSyncChatAsync(acctID, chatID int, markSeen bool) {
	if acctID <= 0 || chatID <= 0 || m.Client == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ids, err := m.Client.GetMessageIDs(ctx, acctID, chatID)
	if err != nil {
		return
	}
	win := history.NewLatest(ids, 0, history.DefaultBatchSize)
	visible := win.VisibleIDs()
	if len(visible) == 0 {
		m.EnqueueUpdate(func(model *Model) {
			if model.ActiveAcctID != acctID {
				return
			}
			model.Windows[chatID] = win
			model.Messages[chatID] = nil
		})
		return
	}

	// Delta optimization: snapshot already-cached IDs and only fetch missing IDs
	// plus the last 5 tail messages (for read/delivered/edit freshness).
	existingCh := make(chan map[int]bool, 1)
	m.EnqueueUpdate(func(model *Model) {
		seen := make(map[int]bool, len(model.Messages[chatID]))
		for _, msg := range model.Messages[chatID] {
			seen[msg.ID] = true
		}
		existingCh <- seen
	})
	var cachedIDs map[int]bool
	select {
	case cachedIDs = <-existingCh:
	case <-time.After(150 * time.Millisecond):
	}

	fetchIDs := make([]int, 0, len(visible))
	tailStart := max(len(visible)-5, 0)
	for i, id := range visible {
		if len(cachedIDs) == 0 || !cachedIDs[id] || i >= tailStart {
			fetchIDs = append(fetchIDs, id)
		}
	}
	if len(fetchIDs) == 0 {
		return
	}
	deltaMsgs, err := m.Client.GetMessages(ctx, acctID, fetchIDs)
	if err != nil {
		return
	}
	if markSeen {
		var unseen []int
		for _, msg := range deltaMsgs {
			if !msg.IsOutgoing && !msg.IsInfo && msg.State < dc.MsgStateInSeen {
				unseen = append(unseen, msg.ID)
			}
		}
		if len(unseen) > 0 {
			_ = m.Client.MarkSeenMsgs(ctx, acctID, unseen)
		}
	}
	visibleSet := make(map[int]bool, len(visible))
	for _, id := range visible {
		visibleSet[id] = true
	}
	m.EnqueueUpdate(func(model *Model) {
		if model.ActiveAcctID != acctID {
			return
		}
		model.Windows[chatID] = win
		merged := mergeDeltaMessages(model.Messages[chatID], deltaMsgs)
		filtered := make([]dc.Message, 0, len(merged))
		for _, msg := range merged {
			if visibleSet[msg.ID] {
				filtered = append(filtered, msg)
			}
		}
		model.Messages[chatID] = filtered
		model.saveCacheAsync()
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
		previewText := fmt.Sprintf("Sent %d attachments", len(files))
		if len(files) == 1 {
			previewText = "Attachment: " + files[0].Name
		}
		if text != "" {
			previewText = text
		}
		m.updateChatPreview(chatID, previewText, now)
		return
	}

	acctID := m.ActiveAcctID
	safeGo("send_draft", func() {
		if len(files) == 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
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
				fctx, fcancel := context.WithTimeout(context.Background(), 60*time.Second)
				_, err := m.Client.SendFileMessage(fctx, acctID, chatID, att.Path, caption)
				fcancel()
				if err != nil {
					m.EnqueueUpdate(func(model *Model) {
						model.Toaster.Show("Upload failed", att.Name+": "+err.Error(), ui.ToastError)
					})
					break
				}
			}
		}
		m.loadChatMessagesAsync(acctID, chatID)
	})
}

func (m *Model) updateChatPreview(chatID int, previewText string, ts int64) {
	for i := range m.Chats {
		if m.Chats[i].ID == chatID {
			m.Chats[i].LastMessage = previewText
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

// CreateDemoChat creates a new local chat entry.
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

// StartLiveSync loads configured accounts & chatlist and runs the live event loop.
func (m *Model) StartLiveSync(ctx context.Context) {
	if m.Client == nil {
		return
	}
	safeGo("live_sync", func() {
		accounts, err := m.Client.GetAllAccounts(ctx)
		if err != nil {
			m.EnqueueUpdate(func(model *Model) {
				model.StatusText = "RPC error: " + err.Error()
			})
			return
		}
		// Hydrate display names & addresses and pick the first Configured profile.
		chosenID := 0
		for i := range accounts {
			if accounts[i].DisplayName == "" {
				if dn, err := m.Client.GetConfig(ctx, accounts[i].ID, "displayname"); err == nil && dn != "" {
					accounts[i].DisplayName = dn
				}
			}
			if accounts[i].Addr == "" {
				if addr, err := m.Client.GetConfig(ctx, accounts[i].ID, "addr"); err == nil && addr != "" {
					accounts[i].Addr = addr
				}
			}
			if accounts[i].DisplayName == "" {
				accounts[i].DisplayName = fmt.Sprintf("Account #%d", accounts[i].ID)
			}
			if chosenID == 0 && accounts[i].Kind == "Configured" {
				chosenID = accounts[i].ID
			}
		}
		if chosenID == 0 && len(accounts) > 0 {
			chosenID = accounts[0].ID
		}

		var chats []dc.ChatEntry
		var initialChatID int
		var initialWin history.Window
		var initialMsgs []dc.Message

		if chosenID > 0 {
			_ = m.Client.SelectAccount(ctx, chosenID)
			// Wake up IMAP IDLE & SMTP immediately in parallel at t=0 (never blocks first-paint hydration).
			go func() {
				_ = m.Client.StartIOForAllAccounts(ctx)
				_ = m.Client.MaybeNetwork(ctx)
			}()
			// Fast path: load chatlist + first chat concurrently with network wake-up.
			ids, _ := m.Client.GetChatlistEntries(ctx, chosenID, nil, "")
			chats, _ = m.Client.GetChatlistItems(ctx, chosenID, ids)
			if len(chats) > 0 {
				initialChatID = chats[0].ID
				if msgIDs, err := m.Client.GetMessageIDs(ctx, chosenID, initialChatID); err == nil {
					initialWin = history.NewLatest(msgIDs, 0, history.DefaultBatchSize)
					initialMsgs, _ = m.Client.GetMessages(ctx, chosenID, initialWin.VisibleIDs())
				}
			}
		}

		m.EnqueueUpdate(func(model *Model) {
			model.Accounts = accounts
			model.ActiveAcctID = chosenID
			model.Chats = chats
			if initialChatID > 0 {
				model.Windows[initialChatID] = initialWin
				model.Messages[initialChatID] = initialMsgs
				if model.ActiveChatID == 0 {
					model.ActiveChatID = initialChatID
				}
			}
			model.saveCacheAsync()
		})

		// Non-blocking background startup: SecureJoin QR & top-5 chat prefetch (IO already started at t=0).
		if chosenID > 0 {
			go func(acctID int, chatList []dc.ChatEntry, skipChat int) {
				if qr, err := m.Client.GetChatSecurejoinQRCode(ctx, acctID, 0); err == nil && qr != "" {
					m.EnqueueUpdate(func(model *Model) {
						model.InviteLink = qr
						model.saveCacheAsync()
					})
				}
				prefetched := 0
				for _, ch := range chatList {
					if ch.ID == skipChat {
						continue
					}
					m.deltaSyncChatAsync(acctID, ch.ID, false)
					prefetched++
					if prefetched >= 5 {
						break
					}
				}
			}(chosenID, chats, initialChatID)
		}

		// Decoupled async coalescing sync worker: GetNextEvent NEVER blocks on RPC calls!
		eventCh := make(chan dc.Event, 256)
		safeGo("delta_worker", func() { m.runDeltaSyncWorker(ctx, eventCh) })

		for {
			ev, err := m.Client.GetNextEvent(ctx)
			if err != nil {
				close(eventCh)
				return
			}
			if m.OnEvent != nil {
				m.OnEvent(ev)
			}
			switch ev.Event.Kind {
			case "IncomingMsg", "MsgsChanged", "MsgDelivered", "MsgRead", "ReactionsChanged":
				select {
				case eventCh <- ev:
				default:
				}
			}
		}
	})
}


// runDeltaSyncWorker drains queued core events asynchronously, coalescing rapid bursts
// within 35ms while dispatching IncomingMsg immediately for sub-second sync & notifications.
func (m *Model) runDeltaSyncWorker(ctx context.Context, eventCh <-chan dc.Event) {
	type chatBatch struct {
		acctID      int
		chatID      int
		msgIDs      map[int]bool
		hasIncoming bool
		fullReload  bool
	}
	pending := make(map[int]*chatBatch)
	var timer *time.Timer
	var timerC <-chan time.Time

	flush := func() {
		if len(pending) == 0 {
			return
		}
		batches := pending
		pending = make(map[int]*chatBatch)
		for _, b := range batches {
			batch := b
			safeGo("delta_batch", func() { m.applyCoalescedBatch(ctx, batch.acctID, batch.chatID, batch.msgIDs, batch.hasIncoming, batch.fullReload) })
		}
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-timerC:
			timerC = nil
			flush()
		case ev, ok := <-eventCh:
			if !ok {
				flush()
				return
			}
			activeAcct := m.currentAccountID()
			if activeAcct <= 0 || ev.ContextID != activeAcct {
				continue
			}
			// Priority fast-path: IncomingMsg fires immediately without waiting for debounce window!
			if ev.Event.Kind == "IncomingMsg" && ev.Event.MsgID > 0 && ev.Event.ChatID > 0 {
				go m.applyCoalescedBatch(ctx, activeAcct, ev.Event.ChatID, map[int]bool{ev.Event.MsgID: true}, true, false)
				continue
			}
			b, exists := pending[ev.Event.ChatID]
			if !exists {
				b = &chatBatch{acctID: activeAcct, chatID: ev.Event.ChatID, msgIDs: make(map[int]bool)}
				pending[ev.Event.ChatID] = b
			}
			if ev.Event.MsgID > 0 {
				b.msgIDs[ev.Event.MsgID] = true
			} else {
				b.fullReload = true
			}
			if timerC == nil {
				if timer == nil {
					timer = time.NewTimer(35 * time.Millisecond)
				} else {
					timer.Reset(35 * time.Millisecond)
				}
				timerC = timer.C
			}
		}
	}
}

func (m *Model) applyCoalescedBatch(ctx context.Context, acctID, chatID int, msgIDSet map[int]bool, isIncoming, fullReload bool) {
	if acctID <= 0 || m.Client == nil {
		return
	}
	if chatID <= 0 || fullReload {
		chatIDs, err := m.Client.GetChatlistEntries(ctx, acctID, nil, "")
		if err == nil {
			updatedChats, _ := m.Client.GetChatlistItems(ctx, acctID, chatIDs)
			m.EnqueueUpdate(func(model *Model) {
				model.Chats = updatedChats
				model.saveCacheAsync()
				if chatID > 0 && chatID == model.ActiveChatID {
					go model.deltaSyncChatAsync(acctID, chatID, true)
				}
			})
		}
		return
	}

	msgIDs := make([]int, 0, len(msgIDSet))
	for id := range msgIDSet {
		msgIDs = append(msgIDs, id)
	}

	// Parallel RPC fetch: messages and chat summary concurrently!
	var deltaMsgs []dc.Message
	var deltaChats []dc.ChatEntry
	var wg sync.WaitGroup
	if len(msgIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			deltaMsgs, _ = m.Client.GetMessages(ctx, acctID, msgIDs)
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		deltaChats, _ = m.Client.GetChatlistItems(ctx, acctID, []int{chatID})
	}()
	wg.Wait()

	m.EnqueueUpdate(func(model *Model) {
		if model.ActiveAcctID != acctID {
			return
		}
		var chatEntry *dc.ChatEntry
		if len(deltaChats) > 0 {
			c := deltaChats[0]
			chatEntry = &c
			model.Chats = upsertChatEntry(model.Chats, c, isIncoming)
		}
		if len(deltaMsgs) > 0 {
			model.Messages[chatID] = mergeDeltaMessages(model.Messages[chatID], deltaMsgs)
			if isIncoming {
				latest := deltaMsgs[len(deltaMsgs)-1]
				isMuted := chatEntry != nil && chatEntry.IsMuted
				if !latest.IsOutgoing && !latest.IsInfo && !isMuted {
					title := latest.DisplaySender()
					if chatEntry != nil && chatEntry.Name != "" && chatEntry.Kind == dc.ChatGroup {
						title = chatEntry.Name + " · " + title
					}
					body := latest.Text
					if body == "" && latest.FileName != "" {
						body = "Sent file: " + latest.FileName
					}
					if chatID != model.ActiveChatID {
						model.Toaster.Show(title, body, ui.ToastInfo)
					}
					if model.Notifier != nil {
						model.Notifier.Push(notify.Item{ChatID: chatID, Title: title, Body: body})
					}
				}
			}
		}
		model.saveCacheAsync()
	})
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

func safeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine_panic_recovered", "worker", name, "panic", fmt.Sprint(r), "stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}

// WakeNetwork nudges deltachat-rpc-server to sync immediately when the window regains focus.
func (m *Model) WakeNetwork() {
	if m == nil || m.DemoMode || m.Client == nil {
		return
	}
	safeGo("wake_network", func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = m.Client.MaybeNetwork(ctx)
	})
}

// Shutdown flushes cached state and closes background helpers cleanly on exit/signal.
func (m *Model) Shutdown() {
	if m == nil {
		return
	}
	_ = m.SaveCache()
	if m.Notifier != nil {
		m.Notifier.Close()
	}
}

// expandDroppedPaths resolves dropped files and top-level non-hidden files inside dropped folders.
func expandDroppedPaths(paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, raw := range paths {
		p := strings.TrimSpace(raw)
		if p == "" {
			continue
		}
		info, err := os.Stat(p)
		if err != nil {
			// Allow virtual demo paths when not on disk
			out = append(out, p)
			continue
		}
		if !info.IsDir() {
			if abs, aerr := filepath.Abs(p); aerr == nil {
				p = abs
			}
			out = append(out, p)
			continue
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || strings.HasPrefix(e.Name(), ".") {
				continue
			}
			out = append(out, filepath.Join(p, e.Name()))
		}
	}
	return out
}

// AddAttachments queues one or more files/images for multi-attachment upload,
// deduplicating already-staged paths and expanding dropped folders.
func (m *Model) AddAttachments(paths []string) int {
	added, _ := m.AddAttachmentsDetailed(paths)
	return added
}

// AddAttachmentsDetailed queues files/images without blocking the UI thread on
// large image decoding, returning (addedCount, skippedDuplicateCount).
func (m *Model) AddAttachmentsDetailed(paths []string) (int, int) {
	expanded := expandDroppedPaths(paths)
	existing := make(map[string]bool, len(m.PendingFiles))
	for _, pf := range m.PendingFiles {
		existing[pf.Path] = true
	}
	added := 0
	duplicates := 0
	var asyncDecodePaths []string
	for i, p := range expanded {
		if p == "" {
			continue
		}
		if existing[p] {
			duplicates++
			continue
		}
		existing[p] = true
		name := filepath.Base(p)
		isImg := IsImagePath(p)
		var sizeBytes int64
		onDisk := false
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			sizeBytes = info.Size()
			onDisk = true
		}
		var bmp gfx.Bitmap
		loadingThumb := false
		if isImg {
			if onDisk {
				if m.Invalidator != nil {
					loadingThumb = true
					asyncDecodePaths = append(asyncDecodePaths, p)
				} else if loaded, ok := LoadBitmap(p, 240); ok {
					bmp = loaded
				} else {
					bmp = DemoGradientBitmap(160, 120, byte(len(m.PendingFiles)+i+1))
				}
			} else {
				bmp = DemoGradientBitmap(160, 120, byte(len(m.PendingFiles)+i+1))
			}
		}
		m.PendingFiles = append(m.PendingFiles, PendingAttachment{
			Path:         p,
			Name:         name,
			Size:         sizeBytes,
			IsImage:      isImg,
			LoadingThumb: loadingThumb,
			Bitmap:       bmp,
		})
		added++
	}
	if len(asyncDecodePaths) > 0 {
		m.decodePendingThumbsAsync(asyncDecodePaths)
	}
	return added, duplicates
}

// decodePendingThumbsAsync decodes staged image thumbnails off the UI thread so
// dropping multiple high-resolution photos never freezes Finder/Explorer or the window.
func (m *Model) decodePendingThumbsAsync(paths []string) {
	for idx, path := range paths {
		p := path
		seed := byte(idx + 1)
		safeGo("thumb_decode", func() {
			bmp, ok := LoadBitmap(p, 240)
			if !ok {
				bmp = DemoGradientBitmap(160, 120, seed)
			}
			m.EnqueueUpdate(func(m *Model) {
				for i := range m.PendingFiles {
					if m.PendingFiles[i].Path == p {
						m.PendingFiles[i].Bitmap = bmp
						m.PendingFiles[i].LoadingThumb = false
					}
				}
			})
		})
	}
}

// PendingTotalBytes returns the sum of known byte sizes for all staged attachments.
func (m *Model) PendingTotalBytes() int64 {
	var total int64
	for _, pf := range m.PendingFiles {
		if pf.Size > 0 {
			total += pf.Size
		}
	}
	return total
}

// OpenPreviewForPending opens the universal in-app preview modal for a staged attachment
// before it is sent.
func (m *Model) OpenPreviewForPending(att PendingAttachment) {
	m.OpenPreviewForMessage(dc.Message{
		FilePath:  att.Path,
		FileName:  att.Name,
		FileBytes: att.Size,
	})
	if m.PreviewDoc.Kind == preview.KindImage && m.PreviewBitmap.Width == 0 && att.Bitmap.Width > 0 {
		m.PreviewBitmap = att.Bitmap
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
// Only repaints on DragOver state transitions (not on every hover mouse move) for silky-smooth UX.
func (m *Model) HandleFileDrop(ev gfx.FileDropEvent) gfx.EventResult {
	switch ev.Phase {
	case gfx.FileDropOver:
		changed := !m.DragOver
		m.DragOver = true
		return gfx.EventResult{Handled: true, Repaint: changed}
	case gfx.FileDropLeave:
		changed := m.DragOver
		m.DragOver = false
		return gfx.EventResult{Handled: true, Repaint: changed}
	case gfx.FileDropDrop:
		m.DragOver = false
		if m.ActiveChatID == 0 && len(m.Chats) > 0 {
			m.SelectChat(m.Chats[0].ID)
		}
		paths := ev.Paths
		if len(paths) == 0 && len(ev.Files) > 0 {
			for _, f := range ev.Files {
				if name := strings.TrimSpace(f.Name()); name != "" {
					paths = append(paths, name)
				}
			}
		}
		if len(paths) > 0 {
			added, dups := m.AddAttachmentsDetailed(paths)
			switch {
			case added > 0 && dups > 0:
				m.Toaster.Show("Files attached", fmt.Sprintf("%d item(s) ready (%d duplicate skipped)", added, dups), ui.ToastInfo)
			case added > 0:
				detail := fmt.Sprintf("%d item(s) ready to send", added)
				if total := m.PendingTotalBytes(); total > 0 {
					detail = fmt.Sprintf("%d item(s) ready · %s total", added, formatBytesShort(total))
				}
				m.Toaster.Show("Files attached", detail, ui.ToastInfo)
			case dups > 0:
				m.Toaster.Show("Already attached", "Dropped file(s) are already in the send queue", ui.ToastInfo)
			default:
				m.Toaster.Show("No files added", "Dropped folder contained no attachable files", ui.ToastInfo)
			}
		}
		return gfx.EventResult{Handled: true, Repaint: true}
	}
	return gfx.EventResult{}
}

// HandleSmartPaste pastes copied files (Finder/Explorer) or screenshots/images from the OS clipboard.
// Returns true if files or an image were handled; returns false for plain text so TextField handles it.
func (m *Model) HandleSmartPaste() bool {
	if files, err := app.ClipboardFiles(); err == nil && len(files) > 0 {
		if m.ActiveChatID == 0 && len(m.Chats) > 0 {
			m.SelectChat(m.Chats[0].ID)
		}
		if added := m.AddAttachments(files); added > 0 {
			m.Toaster.Show("Pasted files", fmt.Sprintf("%d file(s) attached from clipboard", added), ui.ToastInfo)
		}
		return true
	}
	if app.HasClipboardImage() {
		if pngBytes, err := app.ClipboardImagePNG(); err == nil && len(pngBytes) > 0 {
			dir := filepath.Join(os.TempDir(), "parla-viva-paste")
			if m.AccountsPath != "" {
				dir = filepath.Join(m.AccountsPath, "paste")
			}
			_ = os.MkdirAll(dir, 0o700)
			outPath := filepath.Join(dir, fmt.Sprintf("clipboard-%s.png", time.Now().Format("20060102-150405")))
			if err := os.WriteFile(outPath, pngBytes, 0o600); err == nil {
				if m.ActiveChatID == 0 && len(m.Chats) > 0 {
					m.SelectChat(m.Chats[0].ID)
				}
				m.AddAttachments([]string{outPath})
				m.Toaster.Show("Pasted image", filepath.Base(outPath), ui.ToastInfo)
				return true
			}
		}
	}
	return false
}

// HandleEscape dismisses the topmost modal, search bar, or active reply/edit/select mode.
func (m *Model) HandleEscape() bool {
	switch {
	case m.DragOver:
		m.DragOver = false
	case m.PreviewOpen:
		m.PreviewOpen = false
	case m.WebxdcOpen:
		m.WebxdcOpen = false
	case m.QuickOpen:
		m.QuickOpen = false
	case m.InviteOpen:
		m.InviteOpen = false
	case m.SettingsOpen:
		m.SettingsOpen = false
	case m.ChatInfoOpen:
		m.ChatInfoOpen = false
	case m.GalleryOpen:
		m.GalleryOpen = false
	case m.AccountsOpen:
		m.AccountsOpen = false
	case m.NewChatOpen:
		m.NewChatOpen = false
	case m.SearchOpen:
		m.SearchOpen = false
		m.SearchQuery = ""
	case m.EditingMsgID > 0:
		m.EditingMsgID = 0
		m.Draft = ""
	case m.ReplyTo != nil:
		m.ReplyTo = nil
	case m.Selecting:
		m.Selecting = false
		m.SelectedMsgs = make(map[int]bool)
	default:
		return false
	}
	return true
}

// SelectRelativeChat moves to the previous (-1) or next (+1) chat in the filtered sidebar list.
func (m *Model) SelectRelativeChat(delta int) {
	chats := m.FilteredChats()
	if len(chats) == 0 {
		return
	}
	idx := 0
	for i, ch := range chats {
		if ch.ID == m.ActiveChatID {
			idx = i
			break
		}
	}
	next := (idx + delta + len(chats)) % len(chats)
	m.SelectChat(chats[next].ID)
}

// OpenPreviewForMessage opens the universal in-app preview modal for images, Markdown, JSON, CSV, code, ZIP, vCard, or binary files.
func (m *Model) OpenPreviewForMessage(msg dc.Message) {
	name := msg.FileName
	if name == "" && msg.FilePath != "" {
		name = filepath.Base(msg.FilePath)
	}
	if name == "" {
		return
	}
	if msg.ViewType == "Webxdc" {
		m.OpenWebxdcMessage(msg)
		return
	}

	var doc preview.Document
	if strings.HasPrefix(msg.FilePath, "demo://") {
		doc = demoPreviewDocument(name, msg.FilePath)
	} else if msg.FilePath != "" {
		loaded, err := preview.LoadFile(msg.FilePath)
		if err == nil {
			doc = loaded
		} else {
			doc = preview.Document{Path: msg.FilePath, Name: name, SizeBytes: msg.FileBytes, Kind: preview.DetectKind(name)}
		}
	} else {
		doc = preview.Document{Name: name, SizeBytes: msg.FileBytes, Kind: preview.DetectKind(name)}
	}

	m.PreviewDoc = doc
	m.PreviewBitmap = gfx.Bitmap{}
	m.PreviewZoom = 0
	if doc.Kind == preview.KindImage {
		if bmp, ok := m.MessageBitmap(msg); ok {
			m.PreviewBitmap = bmp
		}
	}
	if doc.Kind == preview.KindMarkdown {
		m.PreviewMD.Source = doc.Text
	}
	if doc.Kind == preview.KindCSV {
		m.PreviewRows = ui.TableRows(doc.Rows)
	}
	m.PreviewOpen = true
}

func demoPreviewDocument(name, uri string) preview.Document {
	switch uri {
	case "demo://release-notes.md":
		raw := "# Parla Viva v0.3.0\n\n- **Unified Telegram Desktop UI**: Clean 2-pane chrome, date dividers, right-click menus.\n- **In-App Previewer**: Direct preview for Images, Markdown, JSON, CSV tables, Code, ZIP & vCard.\n- **Windows Polish**: Hidden RPC subprocess window & `%LOCALAPPDATA%\\parla\\accounts` auto-discovery.\n"
		return preview.ParseBytes(name, []byte(raw), int64(len(raw)), false)
	case "demo://build-metrics.csv":
		raw := "Platform,Arch,PackageSize,Status\nWindows,amd64,2.76 MB,Verified\nmacOS,arm64,2.42 MB,Verified\nLinux,amd64,3.06 MB,Verified\n"
		return preview.ParseBytes(name, []byte(raw), int64(len(raw)), false)
	default:
		return preview.Document{Path: uri, Name: name, Kind: preview.DetectKind(name)}
	}
}

// PickAttachments opens the native OS file picker safely outside the frame paint pass,
// handling cancellation silently and surfacing real OS dialog errors as a Toast.
func (m *Model) PickAttachments(opts app.OpenFileOptions, demoFallback []string) {
	paths, err := app.OpenFiles(opts)
	if err != nil {
		if errors.Is(err, app.ErrCanceled) {
			return
		}
		if m.DemoMode && len(demoFallback) > 0 {
			m.AddAttachments(demoFallback)
			return
		}
		m.Toaster.Show("File picker unavailable", err.Error(), ui.ToastError)
		return
	}
	if len(paths) == 0 {
		return
	}
	if m.ActiveChatID == 0 && len(m.Chats) > 0 {
		m.SelectChat(m.Chats[0].ID)
	}
	added, dups := m.AddAttachmentsDetailed(paths)
	switch {
	case added > 0 && dups > 0:
		m.Toaster.Show("Files attached", fmt.Sprintf("%d item(s) ready (%d duplicate skipped)", added, dups), ui.ToastInfo)
	case added > 0:
		detail := fmt.Sprintf("%d item(s) ready to send", added)
		if total := m.PendingTotalBytes(); total > 0 {
			detail = fmt.Sprintf("%d item(s) ready · %s total", added, formatBytesShort(total))
		}
		m.Toaster.Show("Files attached", detail, ui.ToastInfo)
	case dups > 0:
		m.Toaster.Show("Already attached", "Selected file(s) are already in the send queue", ui.ToastInfo)
	}
}

// OpenSystemPath opens a local file with the OS default app, falling back to
// RevealPath for executables and surfacing errors via Toast instead of failing silently.
func (m *Model) OpenSystemPath(path string) {
	p := strings.TrimSpace(path)
	if p == "" || strings.HasPrefix(p, "demo://") {
		return
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if err := app.OpenPath(p); err != nil {
		if errors.Is(err, app.ErrExecutablePath) {
			if rerr := app.RevealPath(p); rerr == nil {
				m.Toaster.Show("Revealed in folder", "Executable files are shown in folder instead of launched", ui.ToastInfo)
				return
			}
		}
		m.Toaster.Show("Cannot open file", err.Error(), ui.ToastError)
	}
}

// RevealSystemPath highlights a local file in Finder / File Explorer / file manager,
// surfacing errors via Toast instead of failing silently.
func (m *Model) RevealSystemPath(path string) {
	p := strings.TrimSpace(path)
	if p == "" || strings.HasPrefix(p, "demo://") {
		return
	}
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	if err := app.RevealPath(p); err != nil {
		m.Toaster.Show("Cannot reveal file", err.Error(), ui.ToastError)
	}
}
