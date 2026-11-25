package appui

import (
	"fmt"
	"strings"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"
)

// Page renders the GoIM-inspired, GPU-native desktop UI for Parla Viva.
func (m *Model) Page(c ui.C) {
	theme := c.Theme()
	pal := m.Palette
	if pal.Rail.Alpha == 0 {
		_, pal = BuildAppearance(m.DarkMode, m.ColorScheme)
		m.Palette = pal
	}

	// Global keyboard shortcuts
	if ui.Shortcut(c, ui.Primary, "k") {
		m.QuickOpen = !m.QuickOpen
		m.QuickQuery = ""
	}
	if ui.Shortcut(c, ui.Primary, "f") {
		m.Section = SectionChats
		m.SearchOpen = !m.SearchOpen
		m.SearchQuery = ""
		m.SearchMatch = 0
	}
	if ui.Shortcut(c, ui.Primary, "n") {
		m.NewChatOpen = true
	}
	if ui.Shortcut(c, ui.Primary, "i") {
		m.ChatInfoOpen = !m.ChatInfoOpen
	}
	if ui.Shortcut(c, ui.Primary, ",") {
		if m.Section == SectionSettings {
			m.Section = SectionChats
		} else {
			m.Section = SectionSettings
		}
	}
	if ui.Shortcut(c, ui.Primary, "q") {
		if m.OnQuit != nil {
			m.OnQuit()
		}
	}
	if ui.Shortcut(c, 0, "escape") {
		m.HandleEscape()
	}
	if ui.Shortcut(c, ui.Primary, "v") {
		m.HandleSmartPaste()
	}
	if ui.Shortcut(c, ui.Alt, "up") {
		m.SelectRelativeChat(-1)
	}
	if ui.Shortcut(c, ui.Alt, "down") {
		m.SelectRelativeChat(1)
	}
	if ui.Shortcut(c, ui.Primary, "enter") {
		m.SendDraft()
	}

	ui.Stack(c).Fill().Children(func() {
		ui.Column(c).Fill().Background(pal.Rail).Children(func() {
			m.titlebarPane(c)

			ui.Row(c).Fill().Grow(1).MinHeight(0).Children(func() {
				m.navRailPane(c)

				// GoIM-style inset rounded workspace frame
				ui.Row(c).Grow(1).MinWidth(0).FillHeight().PadSides(0, WorkspaceInset, WorkspaceInset, 0).Children(func() {
					ui.Row(c).Fill().Radius(RadiusCard).Clip().Background(pal.Sidebar).Border(1, pal.Border).Children(func() {
						switch m.Section {
						case SectionFiles:
							m.filesWorkspace(c)
						case SectionSettings:
							m.settingsWorkspace(c)
						case SectionContacts:
							m.sidebarPane(c)
							ui.Splitter(c, &m.Split).Mins(SidebarMinWidth, 420).SplitterThickness(1)
							m.contactsWorkspace(c)
						default:
							m.sidebarPane(c)
							ui.Splitter(c, &m.Split).Mins(SidebarMinWidth, 420).SplitterThickness(1)
							m.conversationPane(c)
						}
					})
				})
			})
		})

		// Drag-and-drop overlay
		if m.DragOver {
			backdropAlpha := float32(0.32)
			if m.DarkMode {
				backdropAlpha = 0.56
			}
			targetTitle := "Drop images or files to send"
			if chat := m.ActiveChat(); chat != nil && strings.TrimSpace(chat.Name) != "" {
				targetTitle = "Drop to send to " + chat.Name
			}
			ui.Column(c).Fill().Center().Background(gfx.Color{Alpha: backdropAlpha}).Children(func() {
				ui.Column(c).Width(430).Pad(28).Radius(18).Background(theme.Card).Border(2, theme.Primary).Dashed().Center().Gap(10).Children(func() {
					ui.Row(c).Size(52, 52).Radius(26).Background(pal.Selected).Center().Children(func() {
						ui.Icon(c, IconPaperclip).Size(24, 24).Color(theme.Primary)
					})
					ui.Text(c, targetTitle).FontSize(17.5).Bold().Color(theme.Primary).Ellipsis()
					ui.Text(c, "Images, folders, Markdown, JSON, CSV, code, ZIP & vCard").FontSize(12.5).Muted()
					ui.Row(c).PadX(12).PadY(4).Radius(999).Background(theme.Muted).Children(func() {
						ui.Text(c, "Release to attach · Press Esc to cancel").FontSize(11.5).Muted()
					})
				})
			})
		}
	})

	// Global Modals & Drawers
	m.quickSwitcherModal(c)
	m.inviteModal(c)
	m.accountsModal(c)
	m.chatInfoModal(c)
	m.galleryModal(c)
	m.previewModal(c)
	m.settingsModal(c)
	m.newChatModal(c)
	m.webxdcDrawer(c)
	ui.Toasts(c, &m.Toaster)
}

func (m *Model) titlebarPane(c ui.C) {
	pal := m.Palette
	theme := c.Theme()

	var leading, trailing float32
	if m.CustomTitlebar {
		leading, trailing = app.TitlebarInsets()
	}

	statusLabel, statusColor := m.ConnectivityLabel()

	ui.Row(c).FillWidth().Height(TitlebarHeight).Shrink(0).MovesWindow().
		PadSides(0, trailing+12, 0, leading+16).Items(ui.Center).Gap(8).
		Background(pal.Rail).Children(func() {
		// Brand mark + title
		if m.AppIconBitmap.Width > 0 {
			ui.Image(c, m.AppIconBitmap).Size(22, 22).Radius(5)
		} else {
			ui.Row(c).Size(22, 22).Radius(6).Background(theme.Primary).Center().Shrink(0).Children(func() {
				ui.Icon(c, IconMessage).Size(13, 13).Color(theme.PrimaryForeground)
			})
		}
		ui.Text(c, "Parla Viva").FontSize(14.5).Weight(gfx.WeightSemibold).Color(theme.Foreground).Shrink(0)

		ui.Spacer(c)

		// Live Delta Chat connectivity pill (mirrors GoIM's LAN status pill)
		statusBtn := ui.Row(c).Height(26).PadX(10).Radius(13).Gap(6).Items(ui.Center).Shrink(0).
			Hover(ui.Look{}.Background(pal.Hover)).
			Hit(ui.ID(c), "Connection status")
		statusBtn.Children(func() {
			ui.Row(c).Size(7, 7).Radius(3.5).Background(statusColor).Shrink(0)
			ui.Text(c, statusLabel).FontSize(11.5).Color(pal.SecondaryText)
		})
		if statusBtn.Clicked() {
			if m.DemoMode {
				m.InviteOpen = true
			} else {
				m.WakeNetwork()
				m.Toaster.Show("Delta Chat", "Triggered immediate network sync", ui.ToastInfo)
			}
		}

		// Pin window button when host supports SetAlwaysOnTop
		if m.OnTogglePin != nil {
			pinLabel := "Pin window"
			if m.MainPinned {
				pinLabel = "Unpin window"
			}
			pinBtn := ui.Row(c).Size(28, 28).Radius(7).Center().Shrink(0).
				Hover(ui.Look{}.Background(pal.Hover)).
				Hit(ui.ID(c), pinLabel)
			if m.MainPinned {
				pinBtn = pinBtn.Background(pal.Selected)
			}
			pinBtn.Children(func() {
				col := pal.Icon
				if m.MainPinned {
					col = pal.SelectedText
				}
				ui.Icon(c, IconPin).Size(15, 15).Color(col)
			})
			if pinBtn.Clicked() {
				m.MainPinned = m.OnTogglePin()
			}
		}
	})
}

func (m *Model) navRailPane(c ui.C) {
	pal := m.Palette
	theme := c.Theme()

	acctName := "Parla"
	if acct := m.ActiveAccount(); acct != nil && acct.DisplayName != "" {
		acctName = acct.DisplayName
	}

	unread := m.TotalUnreadCount()
	mutedUnread := m.TotalMutedUnreadCount()
	unreadBadge := ""
	unreadMutedOnly := false
	if unread > 0 {
		if unread > 99 {
			unreadBadge = "99+"
		} else {
			unreadBadge = fmt.Sprintf("%d", unread)
		}
	} else if mutedUnread > 0 {
		unreadMutedOnly = true
		if mutedUnread > 99 {
			unreadBadge = "99+"
		} else {
			unreadBadge = fmt.Sprintf("%d", mutedUnread)
		}
	}

	ui.Column(c).Width(RailWidth).Shrink(0).FillHeight().
		PadSides(8, 6, 12, 6).Items(ui.Center).Gap(5).
		Background(pal.Rail).Children(func() {
		// Top Account Avatar (click to open Accounts switcher)
		avatarBtn := ui.Stack(c).Size(42, 42).Shrink(0).Hit(ui.ID(c), "Accounts")
		avatarBtn.Children(func() {
			m.renderAvatar(c, acctName, "", 42, !m.DemoMode || len(m.Accounts) > 0, pal.Rail)
		})
		if avatarBtn.Clicked() {
			m.AccountsOpen = true
		}

		ui.Row(c).Height(6)

		if m.railButton(c, IconMessage, "Chats", "Chats", m.Section == SectionChats, unreadBadge, unreadMutedOnly).Clicked() {
			m.Section = SectionChats
		}
		if m.railButton(c, IconUsers, "Contacts", "Contacts", m.Section == SectionContacts, "", false).Clicked() {
			m.Section = SectionContacts
		}
		if m.railButton(c, IconFolder, "Files", "Files", m.Section == SectionFiles, "", false).Clicked() {
			m.Section = SectionFiles
		}

		ui.Spacer(c)

		// Bottom utility actions (matching GoIM's Lock / Theme / Settings stack)
		if m.railButton(c, IconUserPlus, "Pair", "Invite / Pair", m.InviteOpen, "", false).Clicked() {
			m.InviteOpen = true
		}

		themeIcon := IconMoon
		themeAccessible := "Dark"
		if m.DarkMode {
			themeIcon = IconSun
			themeAccessible = "Light"
		}
		if m.railButton(c, themeIcon, "Theme", themeAccessible, false, "", false).Clicked() {
			m.SetDarkMode(!m.DarkMode)
		}

		if m.railButton(c, IconSettings, "Settings", "Settings", m.Section == SectionSettings, "", false).Clicked() {
			if m.Section == SectionSettings {
				m.Section = SectionChats
			} else {
				m.Section = SectionSettings
			}
		}

		_ = theme
	})
}

func (m *Model) railButton(c ui.C, icon gfx.Icon, visualLabel, accessibleLabel string, active bool, badge string, mutedBadge bool) ui.Box {
	pal := m.Palette
	theme := c.Theme()

	bg := gfx.Color{}
	fg := pal.Icon
	hover := pal.Hover
	if active {
		bg = pal.Selected
		fg = pal.SelectedText
		hover = pal.Selected
	}

	box := ui.Column(c).Size(RailWidth-12, 54).Radius(RadiusControl).
		Items(ui.Center).Justify(ui.Center).Gap(3).
		Background(bg).Hover(ui.Look{}.Background(hover)).
		Hit(ui.ID(c), accessibleLabel)
	box.Children(func() {
		ui.Stack(c).Width(40).Height(24).Children(func() {
			ui.Row(c).Size(20, 20).Absolute().Left(10).Top(2).Center().Children(func() {
				ui.Icon(c, icon).Size(20, 20).Color(fg)
			})
			if badge != "" {
				badgeBg := theme.Primary
				badgeFg := theme.PrimaryForeground
				if mutedBadge {
					badgeBg = pal.BadgeMutedBg
					badgeFg = pal.BadgeMutedFg
				}
				ui.Row(c).MinWidth(17).Height(17).PadX(4).Radius(8.5).Absolute().Right(0).Top(0).
					Background(badgeBg).Center().Children(func() {
					ui.Text(c, badge).FontSize(9.5).Weight(gfx.WeightSemibold).Color(badgeFg)
				})
			}
		})
		ui.Text(c, visualLabel).FontSize(10.5).Weight(gfx.WeightMedium).Color(fg)
	})
	return box
}

func (m *Model) renderAvatar(c ui.C, name, avatarPath string, size float32, online bool, ringColor gfx.Color) {
	pal := m.Palette
	bg, fg := avatarAccentColor(name, m.DarkMode)
	bmp, hasBmp := m.LoadAvatarBitmap(avatarPath)

	ui.Stack(c).Size(size, size).Shrink(0).Children(func() {
		if hasBmp {
			ui.Row(c).Size(size, size).Radius(size / 2).Clip().Children(func() {
				ui.Image(c, bmp).Size(size, size).Radius(size / 2)
			})
		} else {
			fontSize := max(float32(10.5), size*0.34)
			ui.Row(c).Size(size, size).Radius(size / 2).Background(bg).Center().Children(func() {
				ui.Text(c, initialsFrom(name)).FontSize(fontSize).Weight(gfx.WeightSemibold).Color(fg)
			})
		}
		if online {
			dot := max(float32(8), size*0.24)
			ui.Row(c).Absolute().Right(0).Bottom(0).Size(dot, dot).Radius(dot/2).
				Background(pal.Success).Border(1.5, ringColor)
		}
	})
}

func (m *Model) shellIconBtn(c ui.C, label string, icon gfx.Icon, size, glyph float32, active bool) ui.Box {
	pal := m.Palette
	bg := gfx.Color{}
	fg := pal.Icon
	if active {
		bg = pal.Selected
		fg = pal.SelectedText
	}
	b := ui.Row(c).Size(size, size).Radius(8).Center().Shrink(0).
		Background(bg).Hover(ui.Look{}.Background(pal.Hover)).
		Hit(ui.ID(c), label)
	b.Children(func() {
		ui.Icon(c, icon).Size(glyph, glyph).Color(fg)
	})
	return b
}

func renderFileBadge(c ui.C, name, viewType string, dark bool, tile, glyph float32) {
	icon, _, bg, fg := fileBadgeVisual(name, viewType, dark)
	ui.Row(c).Size(tile, tile).Radius(RadiusControl).Shrink(0).Background(bg).Center().Children(func() {
		ui.Icon(c, icon).Size(glyph, glyph).Color(fg)
	})
}

func formatBytesShort(n int64) string {
	switch {
	case n <= 0:
		return ""
	case n < 1024:
		return fmt.Sprintf("%d B", n)
	case n < 1024*1024:
		return fmt.Sprintf("%.1f KB", float64(n)/1024)
	default:
		return fmt.Sprintf("%.1f MB", float64(n)/(1024*1024))
	}
}

func initialsFrom(name string) string {
	return dcInitials(name)
}

func dcInitials(name string) string {
	n := strings.TrimSpace(name)
	if n == "" {
		return "?"
	}
	var out []rune
	take := true
	for _, r := range n {
		if r == ' ' || r == '-' || r == '_' || r == '.' || r == '(' || r == ')' || r == '/' || r == '→' {
			take = true
			continue
		}
		if take {
			out = append(out, r)
			if len(out) == 2 {
				break
			}
			take = false
		}
	}
	if len(out) == 0 {
		return "?"
	}
	return strings.ToUpper(string(out))
}
