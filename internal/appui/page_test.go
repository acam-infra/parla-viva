package appui

import (
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"
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
	if m.Messages[11][before].Text != "Shipping Parla on Viva!" {
		t.Fatalf("last message = %q", m.Messages[11][before].Text)
	}
}

func TestMultiImageUploadAndLightbox(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	// Open Lightbox on inline image message
	clickControl(t, driver, "View preview-desktop.png")
	if !m.LightboxOpen || m.LightboxTitle != "preview-desktop.png" {
		t.Fatalf("LightboxOpen=%v title=%q", m.LightboxOpen, m.LightboxTitle)
	}
	clickControl(t, driver, "Zoom +")
	if m.LightboxZoom <= 1 {
		t.Fatalf("expected LightboxZoom > 1, got %v", m.LightboxZoom)
	}
	clickControl(t, driver, "Close")

	// Queue multiple images via + Photos / AddAttachments and send album
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
	if len(m.PendingFiles) != 0 {
		t.Fatalf("expected PendingFiles cleared, got %d", len(m.PendingFiles))
	}
}

func TestInviteOnboardingAndBackup(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	clickControl(t, driver, "Invite")
	if !m.InviteOpen {
		t.Fatal("expected InviteOpen to be true")
	}
	m.NewAcctName = "Desktop User"
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	clickControl(t, driver, "Create Chatmail Account")
	if len(m.Accounts) != 3 {
		t.Fatalf("expected 3 accounts after creation, got %d", len(m.Accounts))
	}
	clickControl(t, driver, "Close")

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
}

func TestMultiAccountChatInfoGalleryAndMultiSelect(t *testing.T) {
	m := NewDemoModel()
	driver, _ := openDriver(t, m)

	clickControl(t, driver, "Accounts")
	clickControl(t, driver, "Work Profile (dev@chatmail.example)")
	if m.ActiveAcctID != 2 {
		t.Fatalf("ActiveAcctID = %d, want 2", m.ActiveAcctID)
	}

	clickControl(t, driver, "Info")
	clickControl(t, driver, "Mute Chat")
	if !m.ActiveChat().IsMuted {
		t.Fatal("expected active chat to be muted")
	}
	clickControl(t, driver, "Close")

	clickControl(t, driver, "Gallery")
	if len(m.GalleryItems()) == 0 {
		t.Fatal("expected attachment in default chat gallery")
	}
	clickControl(t, driver, "Close")

	before := len(m.Messages[10])
	clickControl(t, driver, "Select")
	clickControl(t, driver, "Select message 1001")
	clickControl(t, driver, "Delete Selected")
	if len(m.Messages[10]) != before-1 {
		t.Fatalf("messages after delete = %d, want %d", len(m.Messages[10]), before-1)
	}
}
