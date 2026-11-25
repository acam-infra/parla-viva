package appui

import (
	"fmt"

	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/dc"
)

// contactsWorkspace renders the GoIM-style Contact / Group Profile Detail view.
func (m *Model) contactsWorkspace(c ui.C) {
	pal := m.Palette
	theme := c.Theme()

	cardSurface := theme.Background
	if m.DarkMode {
		cardSurface = pal.Sidebar
	}

	ui.Column(c).Grow(1).MinWidth(0).FillHeight().Background(pal.Chat).Children(func() {
		// Group detail selected
		if m.SelectedGroupChatID > 0 {
			var group *dc.ChatEntry
			for i := range m.Chats {
				if m.Chats[i].ID == m.SelectedGroupChatID {
					group = &m.Chats[i]
					break
				}
			}
			if group != nil {
				m.renderGroupProfileCard(c, group, cardSurface)
				return
			}
		}

		// Self account detail selected
		if m.SelectedContactID == -1 {
			if acct := m.ActiveAccount(); acct != nil {
				m.renderSelfProfileCard(c, acct, cardSurface)
				return
			}
		}

		// Direct contact detail selected
		var selected *dc.Contact
		for i := range m.Contacts {
			if m.Contacts[i].ID == m.SelectedContactID {
				selected = &m.Contacts[i]
				break
			}
		}
		if selected == nil && len(m.Contacts) > 0 {
			selected = &m.Contacts[0]
		}
		if selected == nil {
			ui.Column(c).Fill().Center().Pad(32).Gap(12).Children(func() {
				ui.Row(c).Size(62, 62).Radius(18).Background(pal.MutedAvatar).Center().Children(func() {
					ui.Icon(c, IconUsers).Size(28, 28).Color(pal.SecondaryText)
				})
				ui.Text(c, "Select a contact or group").FontSize(16.5).Weight(gfx.WeightSemibold)
				ui.Text(c, "Choose someone from the left list to view details or start an encrypted conversation.").
					FontSize(13).Color(pal.SecondaryText).TextCenter()
			})
			return
		}

		m.renderContactProfileCard(c, *selected, cardSurface)
	})
}

func (m *Model) renderContactProfileCard(c ui.C, contact dc.Contact, surface gfx.Color) {
	pal := m.Palette
	theme := c.Theme()

	ui.Scroll(c.KeyInt(contact.ID)).Fill().Scrollbars().Children(func() {
		ui.Row(c).FillWidth().Justify(ui.Center).PadSides(32, 24, 28, 24).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(600).Gap(14).Children(func() {
				ui.Text(c, "Contact Profile").FontSize(16).Weight(gfx.WeightSemibold).Color(theme.Foreground)

				ui.Column(c).FillWidth().Pad(22).Gap(16).Radius(RadiusControl).
					Background(surface).Border(1, pal.Border).Children(func() {
					// Header Identity row
					ui.Row(c).FillWidth().Items(ui.Center).Gap(16).Children(func() {
						m.renderAvatar(c, contact.DisplayLabel(), contact.ProfileImage, 62, false, surface)
						ui.Column(c).Grow(1).MinWidth(0).Gap(6).Children(func() {
							ui.Text(c, contact.DisplayLabel()).FontSize(21).Weight(gfx.WeightSemibold).Color(theme.Foreground).Ellipsis()
							ui.Row(c).Gap(8).Items(ui.Center).Children(func() {
								dotColor := pal.SecondaryText
								presenceText := "Offline"
								if contact.WasSeenRecently {
									dotColor = pal.Success
									presenceText = "Active recently"
								}
								ui.Row(c).Size(7, 7).Radius(3.5).Background(dotColor)
								ui.Text(c, presenceText).FontSize(12).Color(pal.SecondaryText)

								badgeText := "Delta Chat"
								if contact.IsVerified {
									badgeText = "Verified E2EE"
								} else if contact.IsBot {
									badgeText = "Bot"
								}
								ui.Row(c).Height(21).PadX(8).Radius(6).Center().Background(pal.Selected).Children(func() {
									ui.Text(c, badgeText).FontSize(11).Weight(gfx.WeightMedium).Color(pal.SelectedText)
								})
							})
							if contact.Status != "" {
								ui.Wrap(c, contact.Status).FontSize(13).Color(pal.SecondaryText)
							}
						})
					})

					ui.Row(c).FillWidth().Height(1).Background(pal.Border)

					// Contact Info section
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Text(c, "Contact Information").FontSize(14).Weight(gfx.WeightSemibold).Color(theme.Foreground)
						m.profileFactRow(c, "Chatmail / Email", contact.Address)
						m.profileFactRow(c, "Contact ID", fmt.Sprintf("#%d", contact.ID))
					})

					ui.Row(c).FillWidth().Height(1).Background(pal.Border)

					// Encryption section
					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Text(c, "Security & Protocol").FontSize(14).Weight(gfx.WeightSemibold).Color(theme.Foreground)
						enc := "Autocrypt End-to-End Encrypted"
						if contact.IsVerified {
							enc = "Verified SecureJoin (Guaranteed E2EE)"
						}
						m.profileFactRow(c, "Encryption", enc)
						m.profileFactRow(c, "Backend", "Delta Chat / Chatmail JSON-RPC")
					})
				})

				// Primary centered pill CTA button (matching GoIM's "发消息" button)
				ui.Row(c).FillWidth().Justify(ui.Center).PadTop(4).Children(func() {
					btn := ui.ButtonBase(c, "Send Message").MinWidth(148).Height(40).PadX(20).Radius(20).
						Background(theme.Primary).Center().Gap(8).Children(func() {
						ui.Icon(c, IconMessage).Size(16, 16).Color(theme.PrimaryForeground)
						ui.Text(c, "Send Message").FontSize(13.5).Weight(gfx.WeightSemibold).Color(theme.PrimaryForeground)
					})
					if btn.Clicked() {
						ct := contact
						c.After(func() { m.OpenChatWithContact(ct) })
					}
				})
			})
		})
	})
}

func (m *Model) renderGroupProfileCard(c ui.C, group *dc.ChatEntry, surface gfx.Color) {
	pal := m.Palette
	theme := c.Theme()

	ui.Scroll(c.KeyInt(40000+group.ID)).Fill().Scrollbars().Children(func() {
		ui.Row(c).FillWidth().Justify(ui.Center).PadSides(32, 24, 28, 24).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(600).Gap(14).Children(func() {
				ui.Text(c, "Group Profile").FontSize(16).Weight(gfx.WeightSemibold).Color(theme.Foreground)

				ui.Column(c).FillWidth().Pad(22).Gap(16).Radius(RadiusControl).
					Background(surface).Border(1, pal.Border).Children(func() {
					ui.Row(c).FillWidth().Items(ui.Center).Gap(16).Children(func() {
						m.renderAvatar(c, group.Name, group.AvatarPath, 62, false, surface)
						ui.Column(c).Grow(1).MinWidth(0).Gap(5).Children(func() {
							ui.Text(c, group.Name).FontSize(21).Weight(gfx.WeightSemibold).Color(theme.Foreground).Ellipsis()
							ui.Text(c, group.ChatType+" · Delta Chat Encrypted").FontSize(12.5).Color(pal.SecondaryText)
						})
					})

					ui.Row(c).FillWidth().Height(1).Background(pal.Border)

					ui.Column(c).FillWidth().Gap(8).Children(func() {
						ui.Text(c, "Conversation Details").FontSize(14).Weight(gfx.WeightSemibold)
						m.profileFactRow(c, "Chat ID", fmt.Sprintf("#%d", group.ID))
						m.profileFactRow(c, "Messages cached", fmt.Sprintf("%d", len(m.Messages[group.ID])))
						pinVal := "No"
						if group.IsPinned {
							pinVal = "Pinned"
						}
						m.profileFactRow(c, "Pinned", pinVal)
						muteVal := "Active"
						if group.IsMuted {
							muteVal = "Muted"
						}
						m.profileFactRow(c, "Notifications", muteVal)
					})
				})

				ui.Row(c).FillWidth().Justify(ui.Center).PadTop(4).Children(func() {
					btn := ui.ButtonBase(c, "Open Group Chat").MinWidth(156).Height(40).PadX(20).Radius(20).
						Background(theme.Primary).Center().Gap(8).Children(func() {
						ui.Icon(c, IconMessage).Size(16, 16).Color(theme.PrimaryForeground)
						ui.Text(c, "Open Group Chat").FontSize(13.5).Weight(gfx.WeightSemibold).Color(theme.PrimaryForeground)
					})
					if btn.Clicked() {
						id := group.ID
						c.After(func() { m.OpenGroupChat(id) })
					}
				})
			})
		})
	})
}

func (m *Model) renderSelfProfileCard(c ui.C, acct *dc.Account, surface gfx.Color) {
	pal := m.Palette
	theme := c.Theme()

	ui.Scroll(c.Key("self-card")).Fill().Scrollbars().Children(func() {
		ui.Row(c).FillWidth().Justify(ui.Center).PadSides(32, 24, 28, 24).Children(func() {
			ui.Column(c).FillWidth().MaxWidth(600).Gap(14).Children(func() {
				ui.Text(c, "My Profile").FontSize(16).Weight(gfx.WeightSemibold).Color(theme.Foreground)

				ui.Column(c).FillWidth().Pad(22).Gap(16).Radius(RadiusControl).
					Background(surface).Border(1, pal.Border).Children(func() {
					ui.Row(c).FillWidth().Items(ui.Center).Gap(16).Children(func() {
						m.renderAvatar(c, acct.DisplayName, "", 62, true, surface)
						ui.Column(c).Grow(1).MinWidth(0).Gap(5).Children(func() {
							ui.Text(c, acct.DisplayName).FontSize(21).Weight(gfx.WeightSemibold).Color(theme.Foreground).Ellipsis()
							ui.Row(c).Gap(8).Items(ui.Center).Children(func() {
								ui.Row(c).Size(7, 7).Radius(3.5).Background(pal.Success)
								ui.Text(c, acct.Kind).FontSize(12).Color(pal.SecondaryText)
								ui.Row(c).Height(21).PadX(8).Radius(6).Center().Background(pal.Selected).Children(func() {
									ui.Text(c, "Parla Viva").FontSize(11).Weight(gfx.WeightMedium).Color(pal.SelectedText)
								})
							})
						})
					})
					ui.Row(c).FillWidth().Height(1).Background(pal.Border)
					m.profileFactRow(c, "Address", acct.Addr)
					m.profileFactRow(c, "Account ID", fmt.Sprintf("#%d", acct.ID))
					if m.InviteLink != "" {
						m.profileFactRow(c, "SecureJoin Link", m.InviteLink)
					}
				})

				ui.Row(c).FillWidth().Justify(ui.Center).Gap(10).PadTop(4).Children(func() {
					if ui.Button(c, "Copy Invite Link").Small().Outline().Disabled(m.InviteLink == "").Clicked() {
						m.CopyInviteLink()
					}
					if ui.Button(c, "Switch Account").Small().Clicked() {
						m.AccountsOpen = true
					}
				})
			})
		})
	})
}

func (m *Model) profileFactRow(c ui.C, label, value string) {
	if value == "" {
		return
	}
	ui.Column(c).FillWidth().Gap(3).Children(func() {
		ui.Text(c, label).FontSize(12).Color(m.Palette.SecondaryText)
		ui.Text(c, value).FontSize(13.5).Color(c.Theme().Foreground).Ellipsis()
	})
}
