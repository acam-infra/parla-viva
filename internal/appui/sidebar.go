package appui

import (
	"fmt"
	"strings"

	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/dc"
)

func (m *Model) sidebarPane(c ui.C) {
	pal := m.Palette
	sidebarW := m.Split.Pos
	if sidebarW < SidebarMinWidth {
		sidebarW = SidebarDefaultWidth
	} else if sidebarW > SidebarMaxWidth {
		sidebarW = SidebarMaxWidth
	}

	ui.Column(c).Width(sidebarW).MinWidth(SidebarMinWidth).MaxWidth(SidebarMaxWidth).
		Shrink(0).FillHeight().Clip().Background(pal.Sidebar).Children(func() {
		// Top search input + filter pills
		ui.Column(c).FillWidth().Shrink(0).PadSides(12, 10, 6, 10).Gap(8).Children(func() {
			if m.Section == SectionContacts {
				ui.TextField(c, &m.ContactsQuery, "Search contacts", "Search contacts...").
					Icon(IconSearch).FillWidth().Height(36).Radius(RadiusControl)
				m.sidebarContactFilters(c)
			} else {
				ui.TextField(c, &m.SidebarQuery, "Search chats", "Search chats...").
					Icon(IconSearch).FillWidth().Height(36).Radius(RadiusControl)
				m.sidebarChatFilters(c)
			}
		})

		// Scrollable list body
		ui.Scroll(c.Key(string(m.Section))).Grow(1).FillWidth().MinHeight(0).PadSides(2, 8, 4, 8).Gap(2).Scrollbars().Children(func() {
			if m.Section == SectionContacts {
				m.contactsSidebarList(c)
			} else {
				m.chatsSidebarList(c)
			}
		})

		// Pinned GoIM-style bottom action footer
		ui.Row(c).FillWidth().Height(1).Background(pal.Border)
		ui.Row(c).FillWidth().Height(44).Shrink(0).PadX(10).Gap(4).Items(ui.Center).Children(func() {
			if m.shellIconBtn(c, "Invite / Join", IconUserPlus, 34, 18, false).Clicked() {
				m.InviteOpen = true
			}
			if m.shellIconBtn(c, "New", IconPlus, 34, 18, false).Clicked() {
				m.NewChatOpen = true
			}
			ui.Spacer(c)
			if m.shellIconBtn(c, "Sync", IconRefresh, 34, 16, false).Clicked() {
				if m.Section == SectionContacts {
					m.RefreshContacts()
				} else {
					m.WakeNetwork()
					m.Toaster.Show("Sync", "Requested Delta Chat network sync", ui.ToastInfo)
				}
			}
		})
	})
}

func (m *Model) sidebarChatFilters(c ui.C) {
	unreadCount, pinnedCount := 0, 0
	for _, ch := range m.Chats {
		if ch.UnreadCount > 0 {
			unreadCount++
		}
		if ch.IsPinned {
			pinnedCount++
		}
	}
	items := []struct {
		key   string
		label string
	}{
		{"All", fmt.Sprintf("All %d", len(m.Chats))},
		{"Unread", fmt.Sprintf("Unread %d", unreadCount)},
		{"Pinned", fmt.Sprintf("Pinned %d", pinnedCount)},
	}
	ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
		for _, it := range items {
			item := it
			active := m.SidebarFilter == item.key
			bg := gfx.Color{}
			fg := m.Palette.SecondaryText
			if active {
				bg = m.Palette.Selected
				fg = m.Palette.SelectedText
			}
			pill := ui.Row(c.Key("cf-"+item.key)).Height(26).PadX(10).Radius(13).Items(ui.Center).
				Background(bg).Hover(ui.Look{}.Background(m.Palette.Hover)).
				Hit(ui.ID(c), "Filter "+item.key)
			pill.Children(func() {
				ui.Text(c, item.label).FontSize(11.5).Weight(gfx.WeightMedium).Color(fg)
			})
			if pill.Clicked() {
				m.SidebarFilter = item.key
			}
		}
	})
}

func (m *Model) sidebarContactFilters(c ui.C) {
	groupsCount := 0
	for _, ch := range m.Chats {
		if ch.Kind == dc.ChatGroup || ch.Kind == dc.ChatChannel {
			groupsCount++
		}
	}
	items := []struct {
		key   string
		label string
	}{
		{"All", fmt.Sprintf("All %d", len(m.Contacts)+groupsCount)},
		{"Groups", fmt.Sprintf("Groups %d", groupsCount)},
		{"Direct", fmt.Sprintf("Contacts %d", len(m.Contacts))},
	}
	ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
		for _, it := range items {
			item := it
			active := m.ContactsFilter == item.key
			bg := gfx.Color{}
			fg := m.Palette.SecondaryText
			if active {
				bg = m.Palette.Selected
				fg = m.Palette.SelectedText
			}
			pill := ui.Row(c.Key("ctf-"+item.key)).Height(26).PadX(10).Radius(13).Items(ui.Center).
				Background(bg).Hover(ui.Look{}.Background(m.Palette.Hover)).
				Hit(ui.ID(c), "Contacts filter "+item.key)
			pill.Children(func() {
				ui.Text(c, item.label).FontSize(11.5).Weight(gfx.WeightMedium).Color(fg)
			})
			if pill.Clicked() {
				m.ContactsFilter = item.key
			}
		}
	})
}

func (m *Model) chatsSidebarList(c ui.C) {
	pal := m.Palette
	theme := c.Theme()
	chats := m.FilteredChats()
	if len(chats) == 0 {
		ui.Column(c).FillWidth().PadSides(28, 12, 20, 12).Gap(8).Items(ui.Center).Children(func() {
			ui.Row(c).Size(46, 46).Radius(14).Background(pal.MutedAvatar).Center().Children(func() {
				ui.Icon(c, IconMessage).Size(22, 22).Color(pal.SecondaryText)
			})
			ui.Text(c, "No conversations").FontSize(13.5).Weight(gfx.WeightSemibold)
			ui.Text(c, "Pair via SecureJoin or start a new encrypted chat.").FontSize(11.5).Muted().TextCenter()
			if ui.Button(c, "Invite / Join").XSmall().Outline().Clicked() {
				m.InviteOpen = true
			}
		})
		return
	}

	for _, ch := range chats {
		chat := ch
		active := chat.ID == m.ActiveChatID
		bg := gfx.Color{}
		hover := pal.Hover
		if active {
			bg = pal.Selected
			hover = pal.Selected
		}

		row := ui.Row(c.KeyInt(chat.ID)).FillWidth().Height(SidebarConversationHeight).
			PadSides(8, 8, 8, 8).Gap(10).Radius(RadiusCard).
			Items(ui.Center).Background(bg).Hover(ui.Look{}.Background(hover)).
			Hit(ui.ID(c), chat.Name)
		row.Children(func() {
			ring := pal.Sidebar
			if active {
				ring = pal.Selected
			}
			m.renderAvatar(c, chat.Name, chat.AvatarPath, 38, chat.WasSeenRecently, ring)

			ui.Column(c).Grow(1).MinWidth(0).Clip().Gap(3).Justify(ui.Center).Children(func() {
				// Row 1: Chat title + Pin + Mute + right-aligned timestamp (GoIM style)
				ui.Row(c).FillWidth().Gap(5).Items(ui.Center).Children(func() {
					ui.Text(c, chat.Name).FontSize(13.5).Weight(gfx.WeightMedium).Ellipsis().Grow(1)
					if chat.IsPinned {
						ui.Icon(c, IconPin).Size(12, 12).Color(pal.SecondaryText)
					}
					if chat.IsMuted {
						ui.Icon(c, IconBellOff).Size(12, 12).Color(pal.SecondaryText)
					}
					if ts := chat.ShortTimeLabel(); ts != "" {
						ui.Text(c, ts).FontSize(10.5).Color(pal.SecondaryText).Shrink(0)
					}
				})

				// Row 2: Message preview + right-aligned rounded unread badge
				previewLine := chat.LastMessage
				if chat.SummaryPrefix != "" {
					previewLine = chat.SummaryPrefix + ": " + previewLine
				}
				if previewLine == "" {
					previewLine = "No messages yet"
				}
				ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
					ui.Text(c, previewLine).FontSize(11.5).Color(pal.SecondaryText).Ellipsis().Grow(1)
					if chat.UnreadCount > 0 {
						badgeBg := theme.Primary
						badgeFg := theme.PrimaryForeground
						if chat.IsMuted {
							badgeBg = pal.BadgeMutedBg
							badgeFg = pal.SecondaryText
						}
						ui.Row(c).Shrink(0).MinWidth(20).Height(19).PadX(6).Radius(9.5).Center().
							Background(badgeBg).Children(func() {
							ui.Text(c, fmt.Sprintf("%d", chat.UnreadCount)).FontSize(10.5).Weight(gfx.WeightSemibold).Color(badgeFg)
						})
					}
				})
			})
		})

		if row.Clicked() {
			id := chat.ID
			c.After(func() { m.SelectChat(id) })
		}

		ui.ContextMenu(c, row, func(menu *ui.MenuItems) {
			pinLabel := "Pin chat"
			if chat.IsPinned {
				pinLabel = "Unpin chat"
			}
			if menu.Item(pinLabel).Clicked() {
				id := chat.ID
				c.After(func() { m.TogglePinChat(id) })
			}
			muteLabel := "Mute notifications"
			if chat.IsMuted {
				muteLabel = "Unmute notifications"
			}
			if menu.Item(muteLabel).Clicked() {
				id := chat.ID
				c.After(func() { m.ToggleMuteChat(id) })
			}
			if menu.Item("Chat files").Clicked() {
				id := chat.ID
				c.After(func() {
					m.SelectChat(id)
					m.ChatFilesOpen = true
				})
			}
			menu.Separator()
			if menu.Item("Chat info").Clicked() {
				id := chat.ID
				c.After(func() {
					m.SelectChat(id)
					m.ChatInfoOpen = true
				})
			}
		})

		// Subtle inset row separator (matching GoIM's drawSidebarDivider)
		if !active {
			ui.Row(c).FillWidth().Height(1).PadSides(0, 10, 0, 54).Children(func() {
				ui.Row(c).Fill().Background(pal.Border.WithAlpha(0.45))
			})
		}
	}
}

func (m *Model) contactsSidebarList(c ui.C) {
	pal := m.Palette
	q := strings.ToLower(strings.TrimSpace(m.ContactsQuery))

	// Self profile card at top (matching GoIM's selfContactRow)
	if acct := m.ActiveAccount(); acct != nil && (q == "" || strings.Contains(strings.ToLower(acct.DisplayName), q) || strings.Contains(strings.ToLower(acct.Addr), q)) {
		selected := m.SelectedContactID == -1 && m.SelectedGroupChatID == 0
		bg := gfx.Color{}
		if selected {
			bg = pal.Selected
		}
		selfRow := ui.Row(c.Key("self-profile")).FillWidth().Height(SidebarContactHeight).
			PadSides(6, 8, 6, 8).Gap(10).Radius(RadiusCard).Items(ui.Center).
			Background(bg).Hover(ui.Look{}.Background(pal.Hover)).Hit(ui.ID(c), acct.DisplayName+" profile")
		selfRow.Children(func() {
			m.renderAvatar(c, acct.DisplayName, "", 34, true, pal.Sidebar)
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Justify(ui.Center).Children(func() {
				ui.Row(c).Gap(6).Items(ui.Center).Children(func() {
					ui.Text(c, acct.DisplayName).FontSize(13.5).Weight(gfx.WeightMedium).Ellipsis()
					ui.Row(c).Size(6, 6).Radius(3).Background(pal.Success).Shrink(0)
				})
				ui.Text(c, "Self · Delta Chat Profile").FontSize(11).Color(pal.SecondaryText).Ellipsis()
			})
		})
		if selfRow.Clicked() {
			m.SelectedContactID = -1
			m.SelectedGroupChatID = 0
		}
		ui.Row(c).Height(4)
	}

	// Collapsible Group Chats section
	if m.ContactsFilter != "Direct" {
		var groups []dc.ChatEntry
		for _, ch := range m.Chats {
			if ch.Kind != dc.ChatGroup && ch.Kind != dc.ChatChannel {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(ch.Name), q) {
				continue
			}
			groups = append(groups, ch)
		}
		if len(groups) > 0 {
			collapsed := m.CollapsedGroups["groups"] && q == ""
			if m.contactSectionHeader(c, "groups", "My Groups & Channels", fmt.Sprintf("%d", len(groups)), collapsed) {
				m.CollapsedGroups["groups"] = !collapsed
			}
			if !collapsed {
				for _, gr := range groups {
					group := gr
					selected := m.SelectedGroupChatID == group.ID
					bg := gfx.Color{}
					if selected {
						bg = pal.Selected
					}
					row := ui.Row(c.KeyInt(10000+group.ID)).FillWidth().Height(SidebarContactHeight).
						PadSides(6, 8, 6, 8).Gap(10).Radius(RadiusCard).Items(ui.Center).
						Background(bg).Hover(ui.Look{}.Background(pal.Hover)).
						Hit(ui.ID(c), "Group "+group.Name)
					row.Children(func() {
						m.renderAvatar(c, group.Name, group.AvatarPath, 34, group.WasSeenRecently, pal.Sidebar)
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Justify(ui.Center).Children(func() {
							ui.Text(c, group.Name).FontSize(13.5).Weight(gfx.WeightMedium).Ellipsis()
							sub := group.ChatType
							if sub == "" {
								sub = "Group"
							}
							ui.Text(c, sub+" · Encrypted").FontSize(11).Color(pal.SecondaryText).Ellipsis()
						})
					})
					if row.Clicked() {
						m.SelectedGroupChatID = group.ID
						m.SelectedContactID = 0
					}
				}
			}
			ui.Row(c).Height(4)
		}
	}

	// Collapsible Direct Contacts section
	if m.ContactsFilter != "Groups" {
		var filtered []dc.Contact
		onlineCount := 0
		for _, ct := range m.Contacts {
			if q != "" &&
				!strings.Contains(strings.ToLower(ct.DisplayLabel()), q) &&
				!strings.Contains(strings.ToLower(ct.Address), q) {
				continue
			}
			filtered = append(filtered, ct)
			if ct.WasSeenRecently {
				onlineCount++
			}
		}

		collapsed := m.CollapsedGroups["contacts"] && q == ""
		if m.contactSectionHeader(c, "contacts", "Encrypted Contacts", fmt.Sprintf("%d/%d", onlineCount, len(filtered)), collapsed) {
			m.CollapsedGroups["contacts"] = !collapsed
		}
		if !collapsed {
			if len(filtered) == 0 {
				ui.Column(c).FillWidth().Pad(16).Items(ui.Center).Gap(4).Children(func() {
					ui.Text(c, "No matching contacts").FontSize(12.5).Weight(gfx.WeightMedium)
					ui.Text(c, "Use Invite / Pair to add verified peers.").FontSize(11).Muted()
				})
			}
			for _, ct := range filtered {
				contact := ct
				selected := m.SelectedContactID == contact.ID && m.SelectedGroupChatID == 0
				bg := gfx.Color{}
				if selected {
					bg = pal.Selected
				}
				row := ui.Row(c.KeyInt(20000+contact.ID)).FillWidth().Height(SidebarContactHeight).
					PadSides(6, 8, 6, 8).Gap(10).Radius(RadiusCard).Items(ui.Center).
					Background(bg).Hover(ui.Look{}.Background(pal.Hover)).
					Hit(ui.ID(c), "Contact "+contact.DisplayLabel())
				row.Children(func() {
					m.renderAvatar(c, contact.DisplayLabel(), contact.ProfileImage, 34, contact.WasSeenRecently, pal.Sidebar)
					ui.Column(c).Grow(1).MinWidth(0).Gap(2).Justify(ui.Center).Children(func() {
						ui.Row(c).Gap(5).Items(ui.Center).Children(func() {
							ui.Text(c, contact.DisplayLabel()).FontSize(13.5).Weight(gfx.WeightMedium).Ellipsis()
							if contact.IsVerified {
								ui.Icon(c, IconShield).Size(12, 12).Color(pal.Success)
							}
						})
						sub := contact.Address
						if contact.Status != "" {
							sub = contact.Status
						}
						if sub == "" {
							sub = "Delta Chat peer"
						}
						ui.Text(c, sub).FontSize(11).Color(pal.SecondaryText).Ellipsis()
					})
				})
				if row.Clicked() {
					m.SelectedContactID = contact.ID
					m.SelectedGroupChatID = 0
				}
			}
		}
	}
}

func (m *Model) contactSectionHeader(c ui.C, key, title, count string, collapsed bool) bool {
	pal := m.Palette
	icon := IconChevronDown
	if collapsed {
		icon = IconChevronRight
	}
	row := ui.Row(c.Key("sec-"+key)).FillWidth().Height(30).PadX(6).Radius(8).Gap(6).Items(ui.Center).
		Hover(ui.Look{}.Background(pal.Hover)).Hit(ui.ID(c), title)
	row.Children(func() {
		ui.Icon(c, icon).Size(14, 14).Color(pal.SecondaryText)
		ui.Text(c, title).FontSize(12.5).Weight(gfx.WeightSemibold).Ellipsis()
		ui.Spacer(c)
		ui.Text(c, count).FontSize(11).Color(pal.SecondaryText)
	})
	return row.Clicked()
}
