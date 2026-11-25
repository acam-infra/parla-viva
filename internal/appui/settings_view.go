package appui

import (
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/dc"
)

type settingsNavItem struct {
	key   string
	label string
	icon  gfx.Icon
}

var settingsCategories = []settingsNavItem{
	{SettingsTabGeneral, "General & Theme", IconSettings},
	{SettingsTabAccounts, "Accounts & Pairing", IconUsers},
	{SettingsTabNotifications, "Notifications", IconBell},
	{SettingsTabSecurity, "Backup & Security", IconLock},
	{SettingsTabAbout, "About", IconInfo},
}

// settingsWorkspace renders GoIM's full-page categorized Settings workspace.
func (m *Model) settingsWorkspace(c ui.C) {
	pal := m.Palette
	theme := c.Theme()

	ui.Row(c).Fill().Background(pal.Chat).Children(func() {
		// Left category navigation column (GoIM settings sidebar)
		ui.Column(c).Width(228).Shrink(0).FillHeight().Background(pal.Sidebar).
			PadSides(22, 16, 16, 16).Gap(6).Children(func() {
			ui.Text(c, "Settings").FontSize(22).Weight(gfx.WeightSemibold).Color(theme.Foreground)
			ui.Row(c).Height(6)
			for _, cat := range settingsCategories {
				item := cat
				active := m.SettingsTab == item.key
				bg := gfx.Color{}
				fg := theme.Foreground
				if active {
					bg = pal.Selected
					fg = pal.SelectedText
				}
				row := ui.Row(c.Key("stab-"+item.key)).FillWidth().Height(40).PadX(12).Gap(10).
					Radius(RadiusControl).Items(ui.Center).Background(bg).
					Hover(ui.Look{}.Background(pal.Hover)).Hit(ui.ID(c), item.label)
				row.Children(func() {
					ui.Icon(c, item.icon).Size(17, 17).Color(fg)
					ui.Text(c, item.label).FontSize(13.5).Weight(gfx.WeightMedium).Color(fg)
				})
				if row.Clicked() {
					m.SettingsTab = item.key
				}
			}
			ui.Spacer(c)
			if ui.Button(c, "Done").Small().Outline().FillWidth().Clicked() {
				m.Section = SectionChats
			}
		})

		ui.Row(c).Width(1).FillHeight().Background(pal.Border)

		// Right content area
		ui.Column(c).Grow(1).MinWidth(0).FillHeight().PadSides(26, 32, 20, 32).Items(ui.Center).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(680).FillHeight().Gap(16).Children(func() {
				m.settingsHeader(c)
				ui.Scroll(c.Key("settings-body-"+m.SettingsTab)).Grow(1).FillWidth().MinHeight(0).Scrollbars().Children(func() {
					ui.Column(c).FillWidth().Gap(4).PadSides(0, 8, 24, 0).Children(func() {
						switch m.SettingsTab {
						case SettingsTabAccounts:
							m.settingsAccountsContent(c)
						case SettingsTabNotifications:
							m.settingsNotificationsContent(c)
						case SettingsTabSecurity:
							m.settingsSecurityContent(c)
						case SettingsTabAbout:
							m.settingsAboutContent(c)
						default:
							m.settingsGeneralContent(c)
						}
					})
				})
			})
		})
	})
}

func (m *Model) settingsHeader(c ui.C) {
	title, desc := "General & Theme", "Customize appearance, color palette, conversation density, and composer behavior."
	switch m.SettingsTab {
	case SettingsTabAccounts:
		title, desc = "Accounts & Pairing", "Switch Delta Chat profiles, copy your SecureJoin invite link, or provision Chatmail."
	case SettingsTabNotifications:
		title, desc = "Notifications", "Configure desktop alerts and message delivery feedback."
	case SettingsTabSecurity:
		title, desc = "Backup & Security", "Create and verify SafeStore AES-256-GCM encrypted account archives."
	case SettingsTabAbout:
		title, desc = "About Parla Viva", "Delta Chat compatibility, Viva GPU-native engine, and open-source notices."
	}
	ui.Column(c).FillWidth().Gap(4).Children(func() {
		ui.Text(c, title).FontSize(22).Weight(gfx.WeightSemibold).Color(c.Theme().Foreground)
		ui.Text(c, desc).FontSize(12.5).Color(m.Palette.SecondaryText)
	})
}

func (m *Model) settingsRow(c ui.C, title, desc string, control func()) {
	ui.Column(c).FillWidth().Children(func() {
		ui.Row(c).FillWidth().PadY(14).Gap(20).Items(ui.Center).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(3).Children(func() {
				ui.Text(c, title).FontSize(14).Weight(gfx.WeightMedium).Color(c.Theme().Foreground)
				if desc != "" {
					ui.Wrap(c, desc).FontSize(12).Color(m.Palette.SecondaryText)
				}
			})
			if control != nil {
				control()
			}
		})
		ui.Row(c).FillWidth().Height(1).Background(m.Palette.Border)
	})
}

func (m *Model) settingsGeneralContent(c ui.C) {
	m.settingsRow(c, "Dark theme", "Switch between slate dark and clean daylight surfaces while keeping your color scheme.", func() {
		dark := m.DarkMode
		if ui.Switch(c, &dark, "Dark theme").Changed() {
			m.SetDarkMode(dark)
		}
	})

	m.settingsRow(c, "Color scheme", "Curated GoIM-inspired accent palettes for both light and dark modes.", func() {
		scheme := m.ColorScheme
		if ui.Choice(c, &scheme, ColorSchemes, ColorSchemeLabel).Changed() {
			m.SetColorScheme(scheme)
		}
	})

	m.settingsRow(c, "Conversation layout", "Switch between GoIM/Telegram bubbles, Workspace feed, or Compact IRC.", func() {
		ui.Choice(c, &m.Layout, LayoutModes, func(v LayoutMode) string { return string(v) })
	})

	m.settingsRow(c, "Press Enter to send", "When off, Enter inserts a newline and Ctrl/Cmd+Enter sends.", func() {
		ui.Switch(c, &m.EnterToSend, "Press Enter to send")
	})

	// Quick SafeStore Backup shortcut right below general preferences (and full panel in Security tab)
	ui.Row(c).Height(10)
	m.settingsSecurityContent(c)
}

func (m *Model) settingsAccountsContent(c ui.C) {
	pal := m.Palette
	ui.Column(c).FillWidth().Gap(10).PadY(8).Children(func() {
		ui.Text(c, "Configured Delta Chat Profiles").FontSize(14).Weight(gfx.WeightSemibold)
		for _, acct := range m.Accounts {
			a := acct
			label := a.DisplayName
			if a.Addr != "" {
				label += " (" + a.Addr + ")"
			}
			active := a.ID == m.ActiveAcctID
			bg := pal.Sidebar
			if active {
				bg = pal.Selected
			}
			row := ui.Row(c.KeyInt(60000+a.ID)).FillWidth().Pad(12).Radius(RadiusControl).Gap(10).
				Items(ui.Center).Background(bg).Border(1, pal.Border).
				Hover(ui.Look{}.Background(pal.Hover)).Hit(ui.ID(c), label)
			row.Children(func() {
				m.renderAvatar(c, a.DisplayName, "", 34, active, bg)
				ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
					ui.Text(c, a.DisplayName).FontSize(13.5).Weight(gfx.WeightMedium)
					ui.Text(c, a.Addr).FontSize(11.5).Color(pal.SecondaryText)
				})
				if active {
					ui.Text(c, "Active").FontSize(11.5).Weight(gfx.WeightSemibold).Color(pal.SelectedText)
				}
			})
			if row.Clicked() {
				id := a.ID
				c.After(func() { m.SwitchAccount(id) })
			}
		}

		ui.Row(c).Height(6)
		ui.Text(c, "SecureJoin Pairing & Chatmail").FontSize(14).Weight(gfx.WeightSemibold)
		ui.TextField(c, &m.JoinLinkInput, "Peer invite link", "https://i.delta.chat/#... or openpgp4fpr:...").FillWidth()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Join Chat").Small().Clicked() {
				m.JoinViaInviteLink()
			}
			if ui.Button(c, "Copy Invite Link").Small().Outline().Disabled(m.InviteLink == "").Clicked() {
				m.CopyInviteLink()
			}
		})

		ui.Row(c).Height(6)
		ui.Text(c, "Create New Chatmail Profile").FontSize(14).Weight(gfx.WeightSemibold)
		ui.TextField(c, &m.NewAcctName, "Display name", "e.g. Parla Viva Desktop").FillWidth()
		ui.TextField(c, &m.NewAcctRelay, "Chatmail relay", dc.DefaultChatmailRelay).FillWidth()
		if ui.Button(c, "Create Chatmail Account").Small().Outline().Clicked() {
			m.CreateNewChatmailAccount()
		}
		if m.OnboardStatus != "" {
			ui.Wrap(c, m.OnboardStatus).FontSize(12).Color(pal.SecondaryText)
		}
	})
}

func (m *Model) settingsNotificationsContent(c ui.C) {
	m.settingsRow(c, "Desktop notifications", "Show native OS notifications for incoming unmuted Delta Chat messages.", func() {
		ui.Switch(c, &m.NotifyEnabled, "Desktop notifications")
	})
}

func (m *Model) settingsSecurityContent(c ui.C) {
	pal := m.Palette
	ui.Column(c).FillWidth().Pad(16).Radius(RadiusControl).Gap(10).
		Background(pal.Sidebar).Border(1, pal.Border).Children(func() {
		ui.Text(c, "SafeStore Encrypted Backups (AES-256-GCM)").FontSize(14).Weight(gfx.WeightSemibold)
		ui.Wrap(c, "Create a password-protected archive of your Delta Chat accounts directory with SHA-256 manifest verification.").
			FontSize(12).Color(pal.SecondaryText)
		ui.TextField(c, &m.BackupPassword, "Backup password", "Enter strong password...").FillWidth().Secure()
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "Create Backup").Small().Outline().Clicked() {
				m.CreateEncryptedBackup()
			}
			if ui.Button(c, "Verify Backup").Small().Ghost().Disabled(m.LastBackupPath == "").Clicked() {
				m.VerifyLastBackup()
			}
		})
		if m.BackupStatus != "" {
			ui.Wrap(c, m.BackupStatus).FontSize(12).Color(pal.SecondaryText)
		}
	})
}

func (m *Model) settingsAboutContent(c ui.C) {
	pal := m.Palette
	ui.Column(c).FillWidth().Pad(18).Radius(RadiusControl).Gap(10).
		Background(pal.Sidebar).Border(1, pal.Border).Children(func() {
		ui.Text(c, "Parla Viva v0.9.1").FontSize(17).Weight(gfx.WeightSemibold)
		ui.Wrap(c, "GPU-native encrypted Delta Chat / Chatmail desktop client and Agent CLI written in Go on top of vivaui/viva.").
			FontSize(13).Color(pal.SecondaryText)
		ui.Row(c).FillWidth().Height(1).Background(pal.Border)
		m.profileFactRow(c, "Protocol", "Delta Chat JSON-RPC (deltachat-rpc-server) · Parla-compatible accounts")
		m.profileFactRow(c, "UI Design Reference", "Ezreal-byte/GoIM Viva desktop layout & color system (Apache-2.0)")
		m.profileFactRow(c, "License", "Apache License 2.0")
	})
}
