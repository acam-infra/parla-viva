package appui

import (
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/preview"
)

func openDriver(t *testing.T, m *Model) (*app.Driver, *app.Invalidator) {
	t.Helper()
	inv := new(app.Invalidator)
	m.Invalidator = inv
	page := ui.NewPage(ui.PageConfig{
		View:  m.Page,
		Theme: &m.Theme,
	})
	cfg := ui.Configure(page, app.Config{
		Title:       "Parla Viva Test",
		Width:       1200,
		Height:      780,
		Invalidator: inv,
		OnWake:      m.DrainUpdates,
		OnFileDrop:  m.HandleFileDrop,
	})
	driver, err := app.NewDriver(cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(driver.Close)
	return driver, inv
}

func findControl(t *testing.T, driver *app.Driver, name string) gfx.Hit {
	t.Helper()
	for _, hit := range driver.Snapshot().Hits {
		if hit.Label == name && !hit.IsLabel {
			return hit
		}
	}
	t.Fatalf("control %q not found in snapshot", name)
	return gfx.Hit{}
}

func clickControl(t *testing.T, driver *app.Driver, name string) {
	t.Helper()
	b := findControl(t, driver, name).Bounds
	if err := driver.Click(b.X+b.Width/2, b.Y+b.Height/2); err != nil {
		t.Fatalf("click %q: %v", name, err)
	}
}

func TestLiveModelHasNoFakeDemoAccounts(t *testing.T) {
	m := NewLiveModel(nil, nil)
	if len(m.Accounts) != 0 || m.ActiveAcctID != 0 || len(m.Chats) != 0 {
		t.Fatalf("live model leaked demo state: accts=%d active=%d chats=%d", len(m.Accounts), m.ActiveAcctID, len(m.Chats))
	}
}

func TestChatSwitchAndSendMessage(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	clickControl(t, driver, "Elena (Chatmail)")
	if m.ActiveChatID != 11 {
		t.Fatalf("ActiveChatID = %d, want 11", m.ActiveChatID)
	}
	if m.ActiveChat().UnreadCount != 0 {
		t.Fatalf("expected unread count to clear on chat switch, got %d", m.ActiveChat().UnreadCount)
	}

	m.Draft = "Shipping Parla on Viva!"
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	before := len(m.Messages[11])
	clickControl(t, driver, "Send")
	if len(m.Messages[11]) != before+1 {
		t.Fatalf("message count = %d, want %d", len(m.Messages[11]), before+1)
	}
}

func TestCommonFormatInAppPreviewsAndMultiImageUpload(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	// 1. Image Lightbox preview
	clickControl(t, driver, "Preview preview-desktop.png")
	if !m.PreviewOpen || m.PreviewDoc.Kind != preview.KindImage {
		t.Fatalf("expected Image preview, got open=%v kind=%v", m.PreviewOpen, m.PreviewDoc.Kind)
	}
	clickControl(t, driver, "Zoom +")
	if m.PreviewZoom <= 1 {
		t.Fatalf("expected PreviewZoom > 1, got %v", m.PreviewZoom)
	}
	clickControl(t, driver, "Close")

	// 2. Markdown rendered preview
	clickControl(t, driver, "Preview release-notes.md")
	if !m.PreviewOpen || m.PreviewDoc.Kind != preview.KindMarkdown {
		t.Fatalf("expected Markdown preview, got open=%v kind=%v", m.PreviewOpen, m.PreviewDoc.Kind)
	}
	clickControl(t, driver, "Close")

	// 3. CSV interactive Table preview
	clickControl(t, driver, "Preview build-metrics.csv")
	if !m.PreviewOpen || m.PreviewDoc.Kind != preview.KindCSV || len(m.PreviewDoc.Rows) != 3 {
		t.Fatalf("expected CSV preview with 3 rows, got %+v", m.PreviewDoc)
	}
	clickControl(t, driver, "Close")

	// 4. Multi-image album send
	before := len(m.Messages[10])
	m.AddAttachments([]string{"shot-a.png", "shot-b.jpg", "shot-c.png"})
	m.Draft = "3 screenshots"
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	clickControl(t, driver, "Send (3)")
	if len(m.Messages[10]) != before+3 {
		t.Fatalf("messages after multi-image send = %d, want %d", len(m.Messages[10]), before+3)
	}
}

func TestAccountsSettingsBackupAndGallery(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	// Switch account
	clickControl(t, driver, "Accounts")
	clickControl(t, driver, "Work Profile (dev@chatmail.example)")
	if m.ActiveAcctID != 2 {
		t.Fatalf("ActiveAcctID = %d, want 2", m.ActiveAcctID)
	}

	// Open Settings and create/verify SafeStore backup
	clickControl(t, driver, "Settings")
	m.BackupPassword = "test-passphrase-2026"
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	clickControl(t, driver, "Create Backup")
	if m.LastBackupPath == "" || !strings.Contains(m.BackupStatus, "Encrypted backup saved") {
		t.Fatalf("unexpected backup status: %q", m.BackupStatus)
	}
	clickControl(t, driver, "Verify Backup")
	if !strings.Contains(m.BackupStatus, "Verified OK") {
		t.Fatalf("unexpected verify status: %q", m.BackupStatus)
	}
	clickControl(t, driver, "Done")

	// Open Gallery
	clickControl(t, driver, "Gallery")
	if len(m.GalleryItems()) < 3 {
		t.Fatalf("expected >= 3 gallery items, got %d", len(m.GalleryItems()))
	}
	clickControl(t, driver, "Close")
}

func TestDynamicSidebarPinnedAndInteractiveFlows(t *testing.T) {
	m := NewDemoModel()
	// Stress test with 25 chats and ultra-long titles/URLs to ensure sidebar stays strictly pinned.
	for i := 0; i < 20; i++ {
		m.Chats = append(m.Chats, dc.ChatEntry{
			ID:          100 + i,
			Name:        "Long-Conversation-Title-Stress-Test-For-Fixed-Sidebar-Width",
			LastMessage: "https://very-long-url-that-must-not-stretch-the-left-sidebar-pane.example.com/a/b/c/d/e/f",
			ChatType:    "Group",
		})
	}
	driver, inv := openDriver(t, m)

	assertSidebarPinned := func(step string) {
		t.Helper()
		settings := findControl(t, driver, "Settings").Bounds
		accounts := findControl(t, driver, "Accounts").Bounds
		if settings.X+settings.Width > 340 {
			t.Fatalf("[%s] sidebar Settings drifted right: %+v", step, settings)
		}
		if accounts.X+accounts.Width > 340 {
			t.Fatalf("[%s] sidebar Accounts drifted right: %+v", step, accounts)
		}
		if accounts.Y < 700 || accounts.Y+accounts.Height > 780 {
			t.Fatalf("[%s] sidebar Accounts not pinned at bottom of 780px window: %+v", step, accounts)
		}
	}

	assertSidebarPinned("initial")

	// Dynamic flow: switch chat -> toggle dark mode -> open/close search -> send long message -> stage & remove files
	clickControl(t, driver, "Elena (Chatmail)")
	assertSidebarPinned("after-chat-switch")

	clickControl(t, driver, "Dark")
	if !m.DarkMode {
		t.Fatal("expected DarkMode = true")
	}
	assertSidebarPinned("after-dark-mode")

	clickControl(t, driver, "Find")
	if !m.SearchOpen {
		t.Fatal("expected SearchOpen = true")
	}
	clickControl(t, driver, "Close")
	assertSidebarPinned("after-search-toggle")

	m.Draft = strings.Repeat("Long message line without breaking sidebar width. ", 8)
	m.AddAttachments([]string{"report.csv", "diagram.png"})
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	assertSidebarPinned("after-staging-attachments")

	clickControl(t, driver, "Send (2)")
	assertSidebarPinned("after-sending-multi-attachments")
}
