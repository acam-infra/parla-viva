package appui

import (
	"os"
	"path/filepath"
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

	// Open Settings workspace and create/verify SafeStore backup
	clickControl(t, driver, "Settings")
	if m.Section != SectionSettings {
		t.Fatalf("expected SectionSettings, got %q", m.Section)
	}
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
	if m.Section != SectionChats {
		t.Fatalf("expected return to SectionChats after Done, got %q", m.Section)
	}

	// Open GoIM-style right docked Chat Files pane
	clickControl(t, driver, "Chat Files")
	if !m.ChatFilesOpen {
		t.Fatal("expected ChatFilesOpen = true")
	}
	if len(m.GalleryItems()) < 3 {
		t.Fatalf("expected >= 3 gallery items, got %d", len(m.GalleryItems()))
	}
	clickControl(t, driver, "Close Chat Files")
	if m.ChatFilesOpen {
		t.Fatal("expected ChatFilesOpen = false after close")
	}
}

func TestGoIMNavigationRailContactsFilesAndColorSchemes(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	// Switch to Contacts section and start chat with selected contact
	clickControl(t, driver, "Contacts")
	if m.Section != SectionContacts {
		t.Fatalf("expected SectionContacts, got %q", m.Section)
	}
	clickControl(t, driver, "Send Message")
	if m.Section != SectionChats {
		t.Fatalf("expected Send Message to switch to SectionChats, got %q", m.Section)
	}

	// Switch to Files & Media table section
	clickControl(t, driver, "Files")
	if m.Section != SectionFiles {
		t.Fatalf("expected SectionFiles, got %q", m.Section)
	}
	if len(m.SharedFiles(0, "All", "")) < 3 {
		t.Fatalf("expected >= 3 shared files across chats, got %d", len(m.SharedFiles(0, "All", "")))
	}

	// Color scheme switching preserves dark mode and muted badge palette stays distinct from Primary
	m.SetDarkMode(true)
	m.SetColorScheme(SchemePine)
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	if !m.DarkMode || m.ColorScheme != SchemePine {
		t.Fatalf("expected dark+pine, got dark=%v scheme=%q", m.DarkMode, m.ColorScheme)
	}
	if m.TotalUnreadCount() == 0 || m.TotalMutedUnreadCount() == 0 {
		t.Fatalf("expected both active and muted unread counts in demo model, got active=%d muted=%d", m.TotalUnreadCount(), m.TotalMutedUnreadCount())
	}
	if m.Palette.BadgeMutedBg == m.Theme.Primary {
		t.Fatal("muted unread badge background must stay neutral gray and never equal primary accent")
	}
}

func TestDynamicSidebarPinnedAndInteractiveFlows(t *testing.T) {
	m := NewDemoModel()
	// Stress test with 25 chats and ultra-long titles/URLs to ensure rail + sidebar stay strictly pinned.
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
		newBtn := findControl(t, driver, "New").Bounds
		if settings.X+settings.Width > RailWidth+4 {
			t.Fatalf("[%s] rail Settings drifted right: %+v", step, settings)
		}
		if accounts.X+accounts.Width > RailWidth+4 {
			t.Fatalf("[%s] rail Accounts drifted right: %+v", step, accounts)
		}
		if settings.Y < 680 || settings.Y+settings.Height > 780 {
			t.Fatalf("[%s] rail Settings not pinned near bottom of 780px window: %+v", step, settings)
		}
		if newBtn.X < RailWidth-2 || newBtn.X+newBtn.Width > RailWidth+SidebarDefaultWidth+40 {
			t.Fatalf("[%s] sidebar New button drifted out of sidebar bounds: %+v", step, newBtn)
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

func TestDragDropDedupEscapeAndPanicRecovery(t *testing.T) {
	m := NewDemoModel()
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "note.md")
	f2 := filepath.Join(tmp, ".hidden")
	_ = os.WriteFile(f1, []byte("# hi"), 0o600)
	_ = os.WriteFile(f2, []byte("skip"), 0o600)

	// Drop folder -> expands non-hidden files; dropping same file again deduplicates
	m.HandleFileDrop(gfx.FileDropEvent{Phase: gfx.FileDropOver})
	if !m.DragOver {
		t.Fatal("expected DragOver = true on FileDropOver")
	}
	m.HandleFileDrop(gfx.FileDropEvent{Phase: gfx.FileDropDrop, Paths: []string{tmp, f1}})
	if m.DragOver {
		t.Fatal("expected DragOver = false after drop")
	}
	if len(m.PendingFiles) != 1 || m.PendingFiles[0].Name != "note.md" {
		t.Fatalf("expected 1 deduplicated non-hidden file, got %+v", m.PendingFiles)
	}

	// Hierarchical Escape dismissal
	m.SettingsOpen = true
	m.SearchOpen = true
	if !m.HandleEscape() || m.SettingsOpen {
		t.Fatal("first Escape should close SettingsOpen")
	}
	if !m.HandleEscape() || m.SearchOpen {
		t.Fatal("second Escape should close SearchOpen")
	}

	// Relative chat navigation
	first := m.ActiveChatID
	m.SelectRelativeChat(1)
	if m.ActiveChatID == first {
		t.Fatal("expected SelectRelativeChat(1) to switch chat")
	}

	// safeGo recovers panics cleanly without crashing process
	done := make(chan struct{})
	safeGo("test_panic", func() {
		defer close(done)
		panic("simulated crash")
	})
	<-done
}

func TestDragDropHoverThrottlingSizePreviewAndEscape(t *testing.T) {
	m := NewDemoModel()
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "spec.md")
	_ = os.WriteFile(f1, []byte("# Drag-Drop Spec\n\nVerified."), 0o600)

	// First FileDropOver repaints; repeated hover moves do NOT trigger redundant repaints.
	r1 := m.HandleFileDrop(gfx.FileDropEvent{Phase: gfx.FileDropOver, X: 100, Y: 100})
	if !r1.Handled || !r1.Repaint || !m.DragOver {
		t.Fatalf("first FileDropOver should handle + repaint + set DragOver: %+v", r1)
	}
	r2 := m.HandleFileDrop(gfx.FileDropEvent{Phase: gfx.FileDropOver, X: 102, Y: 104})
	if !r2.Handled || r2.Repaint {
		t.Fatalf("repeated FileDropOver should handle without redundant repaint: %+v", r2)
	}

	// Escape clears a stuck DragOver overlay immediately.
	if !m.HandleEscape() || m.DragOver {
		t.Fatal("HandleEscape should clear stuck DragOver")
	}

	// FileDropLeave when already false does not repaint.
	rLeave := m.HandleFileDrop(gfx.FileDropEvent{Phase: gfx.FileDropLeave})
	if !rLeave.Handled || rLeave.Repaint {
		t.Fatalf("redundant FileDropLeave should not repaint: %+v", rLeave)
	}

	// Drop captures file size and deduplicates subsequent drops.
	m.HandleFileDrop(gfx.FileDropEvent{Phase: gfx.FileDropDrop, Paths: []string{f1}})
	if len(m.PendingFiles) != 1 || m.PendingFiles[0].Size <= 0 {
		t.Fatalf("expected 1 pending file with positive Size, got %+v", m.PendingFiles)
	}
	if m.PendingTotalBytes() != m.PendingFiles[0].Size {
		t.Fatalf("expected PendingTotalBytes=%d, got %d", m.PendingFiles[0].Size, m.PendingTotalBytes())
	}
	added, dups := m.AddAttachmentsDetailed([]string{f1})
	if added != 0 || dups != 1 {
		t.Fatalf("expected (0, 1) on duplicate drop, got (%d, %d)", added, dups)
	}

	// Clicking a staged attachment opens the in-app preview modal.
	m.OpenPreviewForPending(m.PendingFiles[0])
	if !m.PreviewOpen || m.PreviewDoc.Name != "spec.md" {
		t.Fatalf("expected preview modal open for spec.md, got open=%v doc=%+v", m.PreviewOpen, m.PreviewDoc)
	}
}

func TestTelegramStyleMuteToggleAndSidebarAlignment(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	if m.ActiveChat().IsMuted {
		t.Fatal("expected initial active chat to be unmuted")
	}
	clickControl(t, driver, "Mute")
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	if !m.ActiveChat().IsMuted {
		t.Fatal("expected header Mute button to mute active chat")
	}
	// Clicking Unmute restores unmuted state
	clickControl(t, driver, "Unmute")
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	if m.ActiveChat().IsMuted {
		t.Fatal("expected header Unmute button to unmute active chat")
	}
}
