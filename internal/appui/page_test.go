package appui

import (
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
		Width:       1100,
		Height:      760,
		Invalidator: inv,
		OnWake:      m.DrainUpdates,
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

	// Switch to Elena's chat (ID 11) and verify unread clears.
	clickControl(t, driver, "Elena (Chatmail)")
	if m.ActiveChatID != 11 {
		t.Fatalf("ActiveChatID = %d, want 11", m.ActiveChatID)
	}
	if m.ActiveChat().UnreadCount != 0 {
		t.Fatalf("expected unread count to clear on chat switch, got %d", m.ActiveChat().UnreadCount)
	}

	// Send a message through the composer.
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

func TestSettingsAndThemeToggle(t *testing.T) {
	m := NewDemoModel()
	driver, inv := openDriver(t, m)

	// Toggle dark mode from sidebar footer.
	clickControl(t, driver, "Dark")
	if !m.DarkMode {
		t.Fatal("expected DarkMode to be true after clicking Dark")
	}

	// Open settings modal and switch layout to Compact IRC.
	clickControl(t, driver, "Settings")
	if !m.SettingsOpen {
		t.Fatal("expected SettingsOpen to be true")
	}
	m.Layout = LayoutCompact
	inv.Invalidate()
	if err := driver.Pump(); err != nil {
		t.Fatal(err)
	}
	clickControl(t, driver, "Done")
	if m.SettingsOpen {
		t.Fatal("expected SettingsOpen to close after clicking Done")
	}
}

func TestWebxdcDrawerAndReactions(t *testing.T) {
	m := NewDemoModel()
	driver, _ := openDriver(t, m)

	// Click +1 reaction on first message.
	clickControl(t, driver, "+1")
	if len(m.Messages[10][0].Reactions) == 0 {
		t.Fatal("expected reaction state on message")
	}

	// Switch to Elena's chat and open Webxdc drawer.
	clickControl(t, driver, "Elena (Chatmail)")
	clickControl(t, driver, "Open Webxdc")
	if !m.WebxdcOpen || m.WebxdcTitle != "checklist.xdc" {
		t.Fatalf("WebxdcOpen=%v title=%q", m.WebxdcOpen, m.WebxdcTitle)
	}
}
