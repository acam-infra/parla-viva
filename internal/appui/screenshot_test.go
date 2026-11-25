package appui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/app/iconpng"
	"github.com/vivaui/viva/headless"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/assets"
)

// TestGoIMStyleScreenshots renders real offscreen PNG screenshots of the
// GoIM-inspired Parla Viva desktop shell when PARLA_VIVA_QA_SCREENSHOTS is set.
func TestGoIMStyleScreenshots(t *testing.T) {
	dir := os.Getenv("PARLA_VIVA_QA_SCREENSHOTS")
	if dir == "" {
		t.Skip("set PARLA_VIVA_QA_SCREENSHOTS to render offscreen QA PNG screenshots")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	render := func(name string, dark bool, scheme string, setup func(*Model)) {
		t.Helper()
		m := NewDemoModel()
		m.CustomTitlebar = true
		if titleIcon, err := iconpng.DecodeFit(assets.AppIconPNG, 48); err == nil && titleIcon.Size > 0 {
			m.AppIconBitmap = BitmapFromRGBA(titleIcon.RGBA, titleIcon.Size, titleIcon.Size)
		}
		m.SetAppearance(dark, scheme)
		if setup != nil {
			setup(m)
		}
		inv := new(app.Invalidator)
		m.Invalidator = inv
		page := ui.NewPage(ui.PageConfig{
			View:  m.Page,
			Theme: &m.Theme,
		})
		cfg := ui.Configure(page, app.Config{
			Title:          "Parla Viva",
			Width:          1360,
			Height:         860,
			MinWidth:       960,
			MinHeight:      620,
			WindowMode:     app.WindowCustomTitlebar,
			TitlebarHeight: TitlebarHeight,
			WindowControls: m.WindowControlsStyle,
			Invalidator:    inv,
			OnWake:         m.DrainUpdates,
			OnFileDrop:     m.HandleFileDrop,
		})
		session, err := headless.Open(cfg, nil, headless.Options{Scale: 1})
		if err != nil {
			t.Fatalf("headless.Open(%s): %v", name, err)
		}
		defer session.Close()
		if err := session.Screenshot(filepath.Join(dir, name)); err != nil {
			t.Fatalf("Screenshot(%s): %v", name, err)
		}
	}

	render("chat-dark.png", true, SchemeClassic, func(m *Model) {
		m.Section = SectionChats
		m.ChatFilesOpen = true
		m.Draft = "Double-checking the SecureJoin invite link before publishing v0.9.1."
	})
	render("chat-light.png", false, SchemeClassic, func(m *Model) {
		m.Section = SectionChats
		m.ChatFilesOpen = true
		m.Draft = "Double-checking the SecureJoin invite link before publishing v0.9.1."
	})
	render("contacts-dark.png", true, SchemeClassic, func(m *Model) {
		m.Section = SectionContacts
		m.SelectedContactID = 2
	})
	render("files-dark.png", true, SchemeClassic, func(m *Model) {
		m.Section = SectionFiles
	})
	render("settings-dark.png", true, SchemeClassic, func(m *Model) {
		m.Section = SectionSettings
		m.SettingsTab = SettingsTabGeneral
	})
	render("chat-pine-dark.png", true, SchemePine, func(m *Model) {
		m.Section = SectionChats
		m.ChatFilesOpen = true
	})
}
