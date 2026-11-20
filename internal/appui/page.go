package appui

import (
	"fmt"
	"strings"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"
	uimarkdown "github.com/vivaui/viva/ui/markdown"
	"github.com/vivaui/viva/ui/message"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/preview"
)

// Page renders the unified Telegram-Desktop-inspired UI for Parla Viva.
func (m *Model) Page(c ui.C) {
	theme := c.Theme()

	// Global keyboard shortcuts
	if ui.Shortcut(c, ui.Primary, "k") {
		m.QuickOpen = !m.QuickOpen
		m.QuickQuery = ""
	}
	if ui.Shortcut(c, ui.Primary, "f") {
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
		m.SettingsOpen = !m.SettingsOpen
	}

	ui.Stack(c).Fill().Children(func() {
		ui.Column(c).Fill().Background(theme.Background).Children(func() {
			if m.DemoMode {
				ui.Row(c).FillWidth().PadX(14).PadY(5).Gap(10).Items(ui.Center).
					Background(theme.Muted).Border(1, theme.Border).Children(func() {
					ui.Text(c, "Offline Preview").FontSize(12).Bold().Color(theme.Primary).Shrink(0)
					ui.Text(c, m.StatusText).FontSize(12).Muted().Grow(1).Ellipsis()
					if ui.Button(c, "Invite / Pair").XSmall().Outline().Clicked() {
						m.InviteOpen = true
					}
				})
			}

			ui.Row(c).Fill().Grow(1).Children(func() {
				m.sidebarPane(c)
				ui.Splitter(c, &m.Split).Mins(260, 420).SplitterThickness(1)
				m.conversationPane(c)
			})
		})

		if m.DragOver {
			ui.Column(c).Fill().Center().Background(gfx.Color{Alpha: 0.45}).Children(func() {
				ui.Column(c).Pad(28).Radius(14).Background(theme.Card).Border(2, theme.Primary).Dashed().Center().Gap(6).Children(func() {
					ui.Text(c, "Drop images or files to send").FontSize(18).Bold().Color(theme.Primary)
					ui.Text(c, "Supports multiple images, Markdown, JSON, CSV, code, ZIP & vCard").FontSize(13).Muted()
				})
			})
		}
	})

	// Modals & Drawers
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

func (m *Model) sidebarPane(c ui.C) {
	theme := c.Theme()
	ui.Column(c).Width(m.Split.Pos).FillHeight().Pad(12).Gap(10).
		Background(theme.Card).Children(func() {
		// Unified clean sidebar header: Title + Actions menu + Settings
		ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
			ui.Text(c, "Parla Viva").FontSize(15.5).Bold().Shrink(0)
			ui.Spacer(c)
			newBtn := ui.Button(c, "+").XSmall().Outline()
			if newBtn.Clicked() {
				m.NewMenuOpen = !m.NewMenuOpen
			}
			ui.Menu(c, newBtn, &m.NewMenuOpen, func(menu *ui.MenuItems) {
				if menu.Item("New Encrypted Chat").Clicked() {
					m.NewChatOpen = true
				}
				if menu.Item("Invite / Join via Link").Clicked() {
					m.InviteOpen = true
				}
				if menu.Item("Quick Switch Chat (Ctrl+K)").Clicked() {
					m.QuickOpen = true
					m.QuickQuery = ""
				}
			})
			if iconBtn(c, "Settings", IconSettings, true).Clicked() {
				m.SettingsOpen = true
			}
		})

		ui.TextField(c, &m.SidebarQuery, "Search chats", "Search...").FillWidth().Radius(8)
		ui.Choice(c, &m.SidebarFilter, ChatFilters, nil)

		chats := m.FilteredChats()
		ui.Scroll(c).Grow(1).Gap(2).Scrollbars().Children(func() {
			if len(chats) == 0 {
				ui.Column(c).FillWidth().PadY(24).Gap(8).Items(ui.Center).Children(func() {
					ui.Text(c, "No conversations").FontSize(13).Muted()
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
				if active {
					bg = theme.Accent
				}
				row := ui.Row(c.KeyInt(chat.ID)).FillWidth().Pad(10).Gap(10).Radius(8).
					Items(ui.Center).Background(bg).
					Hover(ui.Look{}.Background(theme.Hover)).
					Hit(ui.ID(c), chat.Name)
				row.Children(func() {
					avatarBg := theme.Secondary
					avatarFg := theme.SecondaryForeground
					if active {
						avatarBg = theme.Primary
						avatarFg = theme.PrimaryForeground
					}
					ui.Stack(c).Size(40, 40).Shrink(0).Children(func() {
						ui.Row(c).Size(40, 40).Radius(20).Background(avatarBg).Center().Children(func() {
							ui.Text(c, chat.Initials()).FontSize(12).Bold().Color(avatarFg)
						})
						if chat.WasSeenRecently {
							ui.Row(c).Absolute().Right(1).Bottom(1).Size(10, 10).Radius(5).
								Background(theme.Success).Border(1.5, theme.Card)
						}
					})

					ui.Column(c).Grow(1).Gap(2).Children(func() {
						ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
							title := chat.Name
							if chat.IsPinned {
								title = "* " + title
							}
							ui.Text(c, title).FontSize(13.5).Bold().Ellipsis().Grow(1)
							if chat.UnreadCount > 0 {
								ui.Row(c).Shrink(0).MinWidth(22).PadX(6).PadY(2).Radius(10).Center().Background(theme.Primary).Children(func() {
									ui.Text(c, fmt.Sprintf("%d", chat.UnreadCount)).FontSize(11).Bold().Color(theme.PrimaryForeground)
								})
							}
						})
						previewLine := chat.LastMessage
						if chat.SummaryPrefix != "" {
							previewLine = chat.SummaryPrefix + ": " + previewLine
						}
						if previewLine == "" {
							previewLine = "No messages yet"
						}
						ui.Text(c, previewLine).FontSize(12).Muted().Ellipsis()
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
					menu.Separator()
					if menu.Item("Chat info").Clicked() {
						id := chat.ID
						c.After(func() {
							m.SelectChat(id)
							m.ChatInfoOpen = true
						})
					}
				})
			}
		})

		ui.Divider(c)
		ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
			acctLabel := "Setup account"
			if acct := m.ActiveAccount(); acct != nil {
				acctLabel = acct.DisplayName
			}
			if ui.Button(c, "Accounts").XSmall().Outline().Clicked() {
				m.AccountsOpen = true
			}
			ui.Text(c, acctLabel).FontSize(12).Muted().Ellipsis().Grow(1)
			themeLabel := "Dark"
			if m.DarkMode {
				themeLabel = "Light"
			}
			if ui.Button(c, themeLabel).XSmall().Ghost().Clicked() {
				m.SetDarkMode(!m.DarkMode)
			}
		})
	})
}

func (m *Model) conversationPane(c ui.C) {
	theme := c.Theme()
	activeChat := m.ActiveChat()

	ui.Column(c).Grow(1).FillHeight().Children(func() {
		if activeChat == nil {
			ui.Column(c).Fill().Center().Gap(10).Children(func() {
				ui.Text(c, "Select a conversation").FontSize(18).Bold()
				ui.Text(c, "Choose a chat on the left, or pair with Parla using an invite link.").FontSize(13).Muted()
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "Invite / Join Chat").Small().Clicked() {
						m.InviteOpen = true
					}
					if len(m.Accounts) == 0 {
						if ui.Button(c, "Create Chatmail Account").Small().Outline().Clicked() {
							m.InviteOpen = true
						}
					}
				})
			})
			return
		}

		// Unified clean conversation header (Avatar + Title + Find + Gallery + More menu)
		ui.Row(c).FillWidth().PadX(16).PadY(10).Gap(10).Items(ui.Center).
			Background(theme.Card).Border(1, theme.Border).Children(func() {
			ui.Row(c).Size(36, 36).Radius(18).Shrink(0).Background(theme.Secondary).Center().Children(func() {
				ui.Text(c, activeChat.Initials()).FontSize(12).Bold()
			})
			ui.Column(c).Grow(1).Gap(1).Children(func() {
				ui.Text(c, activeChat.Name).FontSize(15).Bold().Ellipsis()
				meta := activeChat.ChatType
				if meta == "" {
					meta = "Encrypted"
				}
				if activeChat.WasSeenRecently {
					meta += " · Active recently"
				}
				if activeChat.IsMuted {
					meta += " · Muted"
				}
				ui.Text(c, meta).FontSize(12).Muted()
			})
			if iconBtn(c, "Find", IconSearch, true).Clicked() {
				m.SearchOpen = !m.SearchOpen
			}
			if iconBtn(c, "Gallery", IconGallery, true).Clicked() {
				m.GalleryOpen = true
			}
			moreBtn := iconBtn(c, "More", IconMore, true)
			if moreBtn.Clicked() {
				m.MoreMenuOpen = !m.MoreMenuOpen
			}
			ui.Menu(c, moreBtn, &m.MoreMenuOpen, func(menu *ui.MenuItems) {
				if menu.Item("Chat Info").Clicked() {
					m.ChatInfoOpen = true
				}
				if menu.Item("Select Messages").Clicked() {
					m.Selecting = !m.Selecting
					if !m.Selecting {
						m.SelectedMsgs = make(map[int]bool)
					}
				}
				menu.Separator()
				if menu.Item("Invite / Pair Link").Clicked() {
					m.InviteOpen = true
				}
			})
		})

		if m.Selecting {
			selectedIDs := m.SelectedMessageIDs()
			ui.Row(c).FillWidth().PadX(16).PadY(8).Gap(10).Items(ui.Center).
				Background(theme.Accent).Children(func() {
				ui.Textf(c, "%d selected", len(selectedIDs)).FontSize(12.5).Bold()
				ui.Spacer(c)
				if ui.Button(c, "Delete Selected").XSmall().Destructive().Disabled(len(selectedIDs) == 0).Clicked() {
					m.DeleteMessageList(activeChat.ID, selectedIDs)
				}
				if ui.Button(c, "Cancel Select").XSmall().Outline().Clicked() {
					m.Selecting = false
					m.SelectedMsgs = make(map[int]bool)
				}
			})
		}

		if pinned := m.PinnedMessages(); len(pinned) > 0 {
			ui.Row(c).FillWidth().PadX(16).PadY(6).Gap(8).Items(ui.Center).
				Background(theme.Muted).Children(func() {
				ui.Text(c, "Pinned").FontSize(11.5).Bold().Color(theme.Primary).Shrink(0)
				ui.Text(c, pinned[0].Text).FontSize(12).Muted().Ellipsis().Grow(1)
			})
		}

		if m.SearchOpen {
			matches := m.SearchMatches()
			ui.Row(c).FillWidth().PadX(16).PadY(8).Gap(8).Items(ui.Center).
				Background(theme.Card).Border(1, theme.Border).Children(func() {
				ui.TextField(c, &m.SearchQuery, "Find in chat", "Search messages...").Grow(1)
				countText := "0 matches"
				if len(matches) > 0 {
					idx := m.SearchMatch % len(matches)
					countText = fmt.Sprintf("%d / %d", idx+1, len(matches))
				}
				ui.Text(c, countText).FontSize(12).Muted().Mono()
				if ui.Button(c, "Next").XSmall().Outline().Disabled(len(matches) == 0).Clicked() {
					m.SearchMatch = (m.SearchMatch + 1) % len(matches)
				}
				if ui.Button(c, "Close").XSmall().Ghost().Clicked() {
					m.SearchOpen = false
					m.SearchQuery = ""
				}
			})
		}

		// Message transcript with Telegram-style date dividers
		msgs := m.Messages[activeChat.ID]
		ui.Scroll(c).Grow(1).FillWidth().Pad(16).Gap(8).Follow(true).Scrollbars().Children(func() {
			if win, ok := m.Windows[activeChat.ID]; ok && win.HasEarlier() {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					if ui.Button(c, "Load earlier messages").XSmall().Outline().Clicked() {
						win.ExpandEarlier(historyDefaultStep)
						m.Windows[activeChat.ID] = win
					}
				})
			}

			lastDate := ""
			for i := range msgs {
				day := msgs[i].DateLabel()
				if day != "" && day != lastDate {
					lastDate = day
					ui.Row(c.Key("day-"+day)).FillWidth().Justify(ui.Center).PadY(4).Children(func() {
						ui.Row(c).PadX(10).PadY(2).Radius(10).Background(theme.Muted).Children(func() {
							ui.Text(c, day).FontSize(11).Muted()
						})
					})
				}
				m.renderMessage(c, msgs[i])
			}
		})

		// Editing / Reply banner
		if m.EditingMsgID > 0 {
			ui.Row(c).FillWidth().PadX(16).PadY(6).Gap(10).Items(ui.Center).
				Background(theme.Muted).Children(func() {
				ui.Text(c, "Editing message").FontSize(12).Bold().Color(theme.Primary)
				ui.Spacer(c)
				if ui.Button(c, "Cancel edit").XSmall().Ghost().Clicked() {
					m.EditingMsgID = 0
					m.Draft = ""
				}
			})
		} else if m.ReplyTo != nil {
			ui.Row(c).FillWidth().PadX(16).PadY(6).Gap(10).Items(ui.Center).
				Background(theme.Muted).Children(func() {
				ui.Text(c, "Replying to "+m.ReplyTo.DisplaySender()+":").FontSize(12).Bold().Color(theme.Primary).Shrink(0)
				ui.Text(c, m.ReplyTo.Text).FontSize(12).Muted().Ellipsis().Grow(1)
				if ui.Button(c, "Cancel reply").XSmall().Ghost().Clicked() {
					m.ReplyTo = nil
				}
			})
		}

		// Multi-image / multi-file attachment preview strip
		if len(m.PendingFiles) > 0 {
			ui.Row(c).FillWidth().PadX(14).PadY(8).Gap(8).Items(ui.Center).
				Background(theme.Muted).Border(1, theme.Border).Children(func() {
				ui.Flow(c).Grow(1).Gap(8).LineGap(6).Children(func() {
					for i := range m.PendingFiles {
						idx := i
						att := m.PendingFiles[idx]
						ui.Row(c.KeyInt(idx)).Pad(6).Gap(6).Radius(8).
							Background(theme.Card).Border(1, theme.Border).Items(ui.Center).Children(func() {
							if att.IsImage && att.Bitmap.Width > 0 {
								ui.Image(c, att.Bitmap).Size(36, 36).Radius(6)
							} else {
								ui.Icon(c, IconFile).Size(16, 16).Color(theme.Primary)
							}
							ui.Text(c, att.Name).FontSize(12).Mono()
							if ui.ButtonBase(c, "Remove attachment").Size(22, 22).Radius(6).Ghost().Center().Children(func() {
								ui.Icon(c, IconClose).Size(12, 12).Color(theme.MutedForeground)
							}).Clicked() {
								c.After(func() { m.RemoveAttachment(idx) })
							}
						})
					}
				})
				if ui.Button(c, "Clear all").XSmall().Ghost().Clicked() {
					m.PendingFiles = nil
				}
			})
		}

		// Unified bottom composer bar with crisp GPU vector icons
		ui.Row(c).FillWidth().PadX(12).PadY(10).Gap(8).Items(ui.Center).
			Background(theme.Card).Border(1, theme.Border).Children(func() {
			if iconBtn(c, "Attach file", IconPaperclip, true).Clicked() {
				opts := app.OpenFileOptions{
					Title:         "Select files to send",
					AllowMultiple: true,
				}
				if paths, err := app.OpenFiles(opts); err == nil && len(paths) > 0 {
					m.AddAttachments(paths)
				} else if m.DemoMode {
					m.AddAttachments([]string{"release-notes.md"})
				}
			}
			if iconBtn(c, "Attach photos", IconImage, true).Clicked() {
				opts := app.OpenFileOptions{
					Title:         "Select images to send",
					AllowMultiple: true,
				}
				if paths, err := app.OpenFiles(opts); err == nil && len(paths) > 0 {
					m.AddAttachments(paths)
				} else if m.DemoMode {
					m.AddAttachments([]string{"photo-1.png", "photo-2.jpg"})
				}
			}
			field := ui.TextField(c, &m.Draft, "Message", "Write a message or drop files...")
			field.Grow(1).Radius(10)
			if m.EnterToSend && field.Submitted() {
				m.SendDraft()
			}
			sendLabel := "Send"
			if m.EditingMsgID > 0 {
				sendLabel = "Save"
			} else if len(m.PendingFiles) > 1 {
				sendLabel = fmt.Sprintf("Send (%d)", len(m.PendingFiles))
			}
			canSend := strings.TrimSpace(m.Draft) != "" || len(m.PendingFiles) > 0
			sendBtn := ui.ButtonBase(c, sendLabel).Height(34).PadX(12).Radius(10).Gap(6).
				Items(ui.Center).Justify(ui.Center).Background(theme.Primary).Disabled(!canSend).
				Children(func() {
					ui.Icon(c, IconSend).Size(15, 15).Color(theme.PrimaryForeground)
					if m.EditingMsgID > 0 {
						ui.Text(c, "Save").FontSize(12.5).Bold().Color(theme.PrimaryForeground)
					} else if len(m.PendingFiles) > 1 {
						ui.Textf(c, "%d", len(m.PendingFiles)).FontSize(12).Bold().Color(theme.PrimaryForeground)
					}
				})
			if sendBtn.Clicked() {
				m.SendDraft()
			}
		})
	})
}


func iconBtn(c ui.C, label string, icon gfx.Icon, ghost bool) ui.Box {
	theme := c.Theme()
	b := ui.ButtonBase(c, label).Size(32, 32).Radius(8).Center().Shrink(0)
	if ghost {
		b = b.Ghost()
	} else {
		b = b.Outline()
	}
	return b.Children(func() {
		ui.Icon(c, icon).Size(16, 16).Color(theme.Foreground)
	})
}

const historyDefaultStep = 30

func (m *Model) renderMessage(c ui.C, msg dc.Message) {
	theme := c.Theme()

	if msg.IsInfo {
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Justify(ui.Center).PadY(4).Children(func() {
			ui.Text(c, msg.Text).FontSize(12).Muted()
		})
		return
	}

	if m.Selecting {
		checked := m.SelectedMsgs[msg.ID]
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Gap(10).Items(ui.Center).Children(func() {
			if ui.Checkbox(c, &checked, fmt.Sprintf("Select message %d", msg.ID)).Changed() {
				m.SelectedMsgs[msg.ID] = checked
			}
			ui.Column(c).Grow(1).Children(func() {
				ui.Text(c, msg.DisplaySender()+": "+msg.Text).FontSize(13.5).Ellipsis()
			})
		})
		return
	}

	if m.Layout == LayoutCompact {
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Gap(8).Items(ui.Start).Children(func() {
			ui.Text(c, msg.TimeLabel()).FontSize(11.5).Muted().Mono()
			ui.Text(c, "<"+msg.DisplaySender()+">").FontSize(13).Bold().Color(theme.Primary)
			ui.Wrap(c, msg.Text).FontSize(13.5).Grow(1)
		})
		return
	}

	if m.Layout == LayoutWorkspace {
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Pad(8).Gap(10).Radius(8).
			Hover(ui.Look{}.Background(theme.Hover)).Children(func() {
			ui.Row(c).Size(28, 28).Radius(6).Shrink(0).Background(theme.Secondary).Center().Children(func() {
				ui.Text(c, initialsFrom(msg.DisplaySender())).FontSize(11).Bold()
			})
			ui.Column(c).Grow(1).Gap(4).Children(func() {
				ui.Row(c).Gap(8).Items(ui.Center).Children(func() {
					ui.Text(c, msg.DisplaySender()).FontSize(13.5).Bold()
					ui.Text(c, msg.TimeLabel()).FontSize(11.5).Muted()
					if msg.IsEdited {
						ui.Text(c, "(edited)").FontSize(11).Muted()
					}
					if status := msg.StatusLabel(); status != "" {
						ui.Text(c, status).FontSize(11).Muted()
					}
				})
				ui.Wrap(c, msg.Text).FontSize(13.5)
				m.reactionPills(c, msg)
			})
		})
		return
	}

	metaTime := msg.TimeLabel()
	if msg.IsEdited {
		metaTime += " · edited"
	}
	if status := msg.StatusLabel(); status != "" {
		metaTime += " · " + status
	}

	row := message.New().
		Name(msg.DisplaySender()).
		Time(metaTime).
		Avatar(initialsFrom(msg.DisplaySender()))

	if msg.IsOutgoing {
		row = row.End().Surface(ui.BubblePrimary)
	} else {
		row = row.Start().Surface(ui.BubbleSecondary)
	}

	bmp, hasBmp := m.MessageBitmap(msg)
	if msg.Quote != nil || msg.ViewType == "Webxdc" || msg.FileName != "" || hasBmp {
		row = row.Content(func(cc ui.C) {
			ui.Column(cc).Gap(6).Pad(10).Radius(12).MaxWidthPercent(75).
				Background(bubbleBg(theme, msg.IsOutgoing)).Children(func() {
				if msg.Quote != nil {
					ui.Column(cc).PadLeft(8).Border(1, theme.Border).Children(func() {
						ui.Text(cc, msg.Quote.AuthorName).FontSize(11.5).Bold()
						ui.Text(cc, msg.Quote.Text).FontSize(12).Muted().Ellipsis()
					})
				}
				if hasBmp {
					imgBox := ui.Row(cc).Size(240, 150).Radius(8).Clip().
						Hit(ui.ID(cc), "Preview "+msg.FileName)
					imgBox.Children(func() {
						ui.Image(cc, bmp).Size(240, 150).Radius(8)
					})
					if imgBox.Clicked() {
						copied := msg
						cc.After(func() { m.OpenPreviewForMessage(copied) })
					}
				}
				fg := theme.SecondaryForeground
				if msg.IsOutgoing {
					fg = theme.PrimaryForeground
				}
				if msg.Text != "" {
					ui.Wrap(cc, msg.Text).FontSize(13.5).Color(fg)
				}
				if msg.ViewType == "Webxdc" {
					ui.Row(cc).Gap(8).Pad(6).Radius(8).Background(theme.Card).Items(ui.Center).Children(func() {
						ui.Text(cc, "Webxdc").FontSize(11).Bold().Color(theme.Primary)
						ui.Text(cc, msg.FileName).FontSize(12).Mono().Grow(1).Ellipsis()
						if ui.Button(cc, "Open Webxdc").XSmall().Outline().Clicked() {
							copied := msg
							cc.After(func() { m.OpenWebxdcMessage(copied) })
						}
					})
				} else if msg.FileName != "" && !hasBmp {
					kindLabel := string(preview.DetectKind(msg.FileName))
					ui.Row(cc).Gap(8).Pad(8).Radius(8).Background(theme.Card).Border(1, theme.Border).Items(ui.Center).Children(func() {
						ui.Text(cc, kindLabel).FontSize(11).Bold().Color(theme.Primary).Shrink(0)
						ui.Text(cc, msg.FileName).FontSize(12).Mono().Grow(1).Ellipsis()
						if msg.FileBytes > 0 {
							ui.Textf(cc, "%d KB", max(msg.FileBytes/1024, 1)).FontSize(11).Muted().Shrink(0)
						}
						if ui.Button(cc, "Preview "+msg.FileName).XSmall().Outline().Clicked() {
							copied := msg
							cc.After(func() { m.OpenPreviewForMessage(copied) })
						}
					})
				}
			})
		})
	} else {
		row = row.Body(msg.Text)
	}

	// Clean Telegram-style footer: ONLY reaction pills (no noisy button spam on every message!)
	if len(msg.Reactions) > 0 {
		row = row.Actions(func(ac ui.C) {
			m.reactionPills(ac, msg)
		})
	}

	box := message.View(c.KeyInt(msg.ID), msg.DisplaySender(), row)
	ui.ContextMenu(c, box, func(menu *ui.MenuItems) {
		if menu.Item("Reply").Clicked() {
			copied := msg
			c.After(func() { m.ReplyTo = &copied })
		}
		if menu.Item("React 👍").Clicked() {
			c.After(func() { m.ToggleReaction(msg.ChatID, msg.ID, "👍") })
		}
		if menu.Item("React ❤️").Clicked() {
			c.After(func() { m.ToggleReaction(msg.ChatID, msg.ID, "❤️") })
		}
		if msg.FileName != "" {
			if menu.Item("Preview attachment").Clicked() {
				copied := msg
				c.After(func() { m.OpenPreviewForMessage(copied) })
			}
		}
		if msg.Text != "" {
			if menu.Item("Copy text").Clicked() {
				_ = app.SetClipboardText(msg.Text)
			}
		}
		pinLabel := "Pin message"
		if msg.IsPinned {
			pinLabel = "Unpin message"
		}
		if menu.Item(pinLabel).Clicked() {
			c.After(func() { m.TogglePinMessage(msg.ChatID, msg.ID) })
		}
		if msg.IsOutgoing {
			if menu.Item("Edit").Clicked() {
				c.After(func() {
					m.EditingMsgID = msg.ID
					m.Draft = msg.Text
				})
			}
		}
		menu.Separator()
		if menu.Item("Delete").Destructive().Clicked() {
			c.After(func() { m.DeleteMessageList(msg.ChatID, []int{msg.ID}) })
		}
	})
}

func (m *Model) reactionPills(c ui.C, msg dc.Message) {
	if len(msg.Reactions) == 0 {
		return
	}
	ui.Row(c).Gap(4).Items(ui.Center).Children(func() {
		for _, rx := range msg.Reactions {
			label := fmt.Sprintf("%s %d", rx.Emoji, rx.Count)
			btn := ui.Button(c.Key("rx-"+rx.Emoji), label).XSmall()
			if rx.BySelf {
				btn = btn.Secondary()
			} else {
				btn = btn.Ghost()
			}
			if btn.Clicked() {
				emoji := rx.Emoji
				c.After(func() { m.ToggleReaction(msg.ChatID, msg.ID, emoji) })
			}
		}
	})
}

// previewModal renders the universal in-app previewer for Images, Markdown, CSV Tables, JSON, Code, ZIP & vCard.
func (m *Model) previewModal(c ui.C) {
	if !m.PreviewOpen {
		return
	}
	doc := m.PreviewDoc
	title := doc.Name
	if title == "" {
		title = "File Preview"
	}
	ui.ModalWith(c, &m.PreviewOpen, title, ui.ModalSpec{Width: 740}, func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			// Metadata header strip
			meta := fmt.Sprintf("%s · %d bytes", doc.Kind, doc.SizeBytes)
			if doc.Truncated {
				meta += " · truncated"
			}
			ui.Row(c).FillWidth().Items(ui.Center).Gap(8).Children(func() {
				ui.Text(c, meta).FontSize(12).Muted().Mono().Grow(1)
				if doc.Text != "" {
					if ui.Button(c, "Copy Content").XSmall().Outline().Clicked() {
						_ = app.SetClipboardText(doc.Text)
						m.Toaster.Show("Copied", doc.Name, ui.ToastSuccess)
					}
				}
			})

			switch doc.Kind {
			case preview.KindImage:
				if m.PreviewBitmap.Width > 0 {
					ui.Slot(c).FillWidth().Height(420).Draw(func(d *ui.DrawContext) {
						components.PaintImageView(d.Canvas, *d.Theme, d.Rect, components.ImageViewOptions{
							Bitmap: m.PreviewBitmap,
							Zoom:   m.PreviewZoom,
						})
					})
					ui.Row(c).FillWidth().Gap(8).Justify(ui.Center).Children(func() {
						if ui.Button(c, "Fit").XSmall().Outline().Clicked() {
							m.PreviewZoom = 0
						}
						if ui.Button(c, "100%").XSmall().Outline().Clicked() {
							m.PreviewZoom = 1
						}
						if ui.Button(c, "Zoom +").XSmall().Outline().Clicked() {
							if m.PreviewZoom == 0 {
								m.PreviewZoom = 1.5
							} else {
								m.PreviewZoom = min(m.PreviewZoom+0.5, 4)
							}
						}
					})
				} else {
					ui.Text(c, "Image file not cached locally yet.").FontSize(13).Muted()
				}

			case preview.KindMarkdown:
				m.PreviewMD.Style = uimarkdown.Style{Size: c.Theme().Metrics.FontSizeM, LineHeight: 22}
				uimarkdown.View(c, "Markdown preview", &m.PreviewMD).FillWidth().Height(380)

			case preview.KindCSV:
				cols := make([]components.TableColumn, len(doc.Headers))
				for i := range cols {
					if i < len(cols)-1 {
						cols[i] = components.TableColumn{Width: 140}
					}
				}
				ui.Table(c, &m.PreviewTable, ui.TableSpec{
					Headers: doc.Headers,
					Source:  &m.PreviewRows,
					Striped: true,
					Columns: cols,
				}).FillWidth().Height(360)

			case preview.KindVCard:
				if doc.VCard != nil {
					ui.Column(c).FillWidth().Pad(14).Radius(10).Gap(6).Background(c.Theme().Muted).Children(func() {
						ui.Text(c, doc.VCard.FullName).FontSize(16).Bold()
						if doc.VCard.Org != "" {
							ui.Text(c, "Organization: "+doc.VCard.Org).FontSize(13).Muted()
						}
						if doc.VCard.Email != "" {
							ui.Text(c, "Email: "+doc.VCard.Email).FontSize(13).Mono()
						}
						if doc.VCard.Phone != "" {
							ui.Text(c, "Phone: "+doc.VCard.Phone).FontSize(13).Mono()
						}
						if doc.VCard.Note != "" {
							ui.Wrap(c, doc.VCard.Note).FontSize(12.5).Muted()
						}
					})
				}

			case preview.KindArchive:
				ui.Scroll(c).FillWidth().Height(340).Gap(4).Scrollbars().Children(func() {
					for i, item := range doc.ArchiveItems {
						ui.Row(c.KeyInt(i)).FillWidth().PadX(8).PadY(4).Gap(8).Children(func() {
							ui.Text(c, item.Name).FontSize(12.5).Mono().Grow(1).Ellipsis()
							ui.Textf(c, "%d B", item.Size).FontSize(11.5).Muted().Mono()
						})
					}
				})

			case preview.KindJSON, preview.KindCode:
				ui.Scroll(c).FillWidth().Height(380).Pad(10).Radius(8).Background(c.Theme().Muted).Scrollbars().Children(func() {
					ui.Wrap(c, doc.Text).FontSize(12.5).Mono()
				})

			default:
				ui.Column(c).FillWidth().Pad(16).Radius(8).Gap(6).Background(c.Theme().Muted).Children(func() {
					ui.Text(c, "Binary attachment preview").FontSize(14).Bold()
					ui.Text(c, "Use Open with System App below to open in your default desktop viewer.").FontSize(13).Muted()
				})
			}
		})
	}, func() {
		if doc.Path != "" && !strings.HasPrefix(doc.Path, "demo://") {
			if ui.Button(c, "Open with System App").Small().Outline().Clicked() {
				_ = app.OpenPath(doc.Path)
			}
			if ui.Button(c, "Reveal in Folder").Small().Ghost().Clicked() {
				_ = app.RevealPath(doc.Path)
			}
		}
		if ui.Button(c, "Close").Clicked() {
			m.PreviewOpen = false
		}
	})
}

func (m *Model) inviteModal(c ui.C) {
	if !m.InviteOpen {
		return
	}
	ui.ModalWith(c, &m.InviteOpen, "Chatmail Account & Invite", ui.ModalSpec{Width: 520}, func() {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, "Pair with Parla / Delta Chat (SecureJoin)").FontSize(13.5).Bold()
			ui.TextField(c, &m.JoinLinkInput, "Peer invite link", "https://i.delta.chat/#... or openpgp4fpr:...").FillWidth()
			if ui.Button(c, "Join Chat").Small().Clicked() {
				m.JoinViaInviteLink()
			}

			ui.Divider(c)
			ui.Text(c, "Your Invite Link").FontSize(13.5).Bold()
			inviteCopy := m.InviteLink
			ui.TextField(c, &inviteCopy, "My invite link", "Not configured yet").FillWidth().ReadOnly(true)
			if ui.Button(c, "Copy Invite Link").Small().Outline().Disabled(m.InviteLink == "").Clicked() {
				m.CopyInviteLink()
			}

			ui.Divider(c)
			ui.Text(c, "Create New Chatmail Account").FontSize(13.5).Bold()
			ui.TextField(c, &m.NewAcctName, "Display name", "e.g. Parla Viva Desktop").FillWidth()
			ui.TextField(c, &m.NewAcctRelay, "Chatmail relay", dc.DefaultChatmailRelay).FillWidth()
			if ui.Button(c, "Create Chatmail Account").Small().Outline().Clicked() {
				m.CreateNewChatmailAccount()
			}
			if m.OnboardStatus != "" {
				ui.Wrap(c, m.OnboardStatus).FontSize(12).Muted()
			}
		})
	}, func() {
		if ui.Button(c, "Close").Clicked() {
			m.InviteOpen = false
		}
	})
}

func (m *Model) quickSwitcherModal(c ui.C) {
	if !m.QuickOpen {
		return
	}
	ui.ModalWith(c, &m.QuickOpen, "Quick Switch Chat", ui.ModalSpec{Width: 420}, func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.TextField(c, &m.QuickQuery, "Jump to chat", "Type a chat name...").FillWidth().AutoFocus()
			q := strings.ToLower(strings.TrimSpace(m.QuickQuery))
			for _, ch := range m.Chats {
				if q != "" && !strings.Contains(strings.ToLower(ch.Name), q) {
					continue
				}
				chat := ch
				if ui.Button(c.KeyInt(chat.ID), chat.Name).Ghost().FillWidth().Clicked() {
					id := chat.ID
					c.After(func() {
						m.SelectChat(id)
						m.QuickOpen = false
					})
				}
			}
		})
	}, nil)
}

func (m *Model) accountsModal(c ui.C) {
	if !m.AccountsOpen {
		return
	}
	ui.ModalWith(c, &m.AccountsOpen, "Switch Account", ui.ModalSpec{Width: 440}, func() {
		ui.Column(c).FillWidth().Gap(8).Children(func() {
			for _, acct := range m.Accounts {
				a := acct
				label := a.DisplayName
				if a.Addr != "" {
					label += " (" + a.Addr + ")"
				}
				btn := ui.Button(c.KeyInt(a.ID), label).FillWidth()
				if a.ID == m.ActiveAcctID {
					btn = btn.Secondary()
				} else {
					btn = btn.Outline()
				}
				if btn.Clicked() {
					id := a.ID
					c.After(func() { m.SwitchAccount(id) })
				}
			}
			ui.Divider(c)
			if ui.Button(c, "+ Add Chatmail Account").Small().Outline().FillWidth().Clicked() {
				m.AccountsOpen = false
				m.InviteOpen = true
			}
		})
	}, func() {
		if ui.Button(c, "Close").Outline().Clicked() {
			m.AccountsOpen = false
		}
	})
}

func (m *Model) chatInfoModal(c ui.C) {
	if !m.ChatInfoOpen {
		return
	}
	chat := m.ActiveChat()
	if chat == nil {
		m.ChatInfoOpen = false
		return
	}
	ui.ModalWith(c, &m.ChatInfoOpen, "Chat Info", ui.ModalSpec{Width: 440}, func() {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, chat.Name).FontSize(16).Bold()
			ui.Textf(c, "Type: %s · Messages loaded: %d", chat.ChatType, len(m.Messages[chat.ID])).FontSize(13).Muted()
			ui.Row(c).Gap(8).Children(func() {
				pinBtn := "Pin Chat"
				if chat.IsPinned {
					pinBtn = "Unpin Chat"
				}
				if ui.Button(c, pinBtn).Small().Outline().Clicked() {
					id := chat.ID
					c.After(func() { m.TogglePinChat(id) })
				}
				muteBtn := "Mute Chat"
				if chat.IsMuted {
					muteBtn = "Unmute Chat"
				}
				if ui.Button(c, muteBtn).Small().Outline().Clicked() {
					id := chat.ID
					c.After(func() { m.ToggleMuteChat(id) })
				}
			})
		})
	}, func() {
		if ui.Button(c, "Close").Clicked() {
			m.ChatInfoOpen = false
		}
	})
}

func (m *Model) galleryModal(c ui.C) {
	if !m.GalleryOpen {
		return
	}
	ui.ModalWith(c, &m.GalleryOpen, "Media & Files Gallery", ui.ModalSpec{Width: 520}, func() {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Tabs(c, &m.GalleryTab, GalleryTabs, nil)
			items := m.GalleryItems()
			if len(items) == 0 {
				ui.Text(c, "No shared media or files in this category.").FontSize(13).Muted()
				return
			}
			for _, item := range items {
				it := item
				ui.Row(c.KeyInt(it.ID)).FillWidth().Pad(8).Gap(10).Radius(8).
					Background(c.Theme().Muted).Items(ui.Center).Children(func() {
					ui.Text(c, string(preview.DetectKind(it.FileName))).FontSize(11.5).Bold().Color(c.Theme().Primary)
					ui.Text(c, it.FileName).FontSize(13).Mono().Grow(1).Ellipsis()
					if ui.Button(c, "Open").XSmall().Outline().Clicked() {
						copied := it
						c.After(func() {
							m.GalleryOpen = false
							m.OpenPreviewForMessage(copied)
						})
					}
				})
			}
		})
	}, func() {
		if ui.Button(c, "Close").Clicked() {
			m.GalleryOpen = false
		}
	})
}

func (m *Model) settingsModal(c ui.C) {
	if !m.SettingsOpen {
		return
	}
	ui.ModalWith(c, &m.SettingsOpen, "Parla Viva Settings", ui.ModalSpec{Width: 480}, func() {
		ui.Column(c).FillWidth().Gap(14).Children(func() {
			ui.Text(c, "Conversation layout").FontSize(13).Bold()
			ui.Choice(c, &m.Layout, LayoutModes, func(v LayoutMode) string { return string(v) })

			ui.Row(c).FillWidth().Items(ui.Center).Children(func() {
				ui.Text(c, "Dark theme").FontSize(13.5)
				ui.Spacer(c)
				dark := m.DarkMode
				if ui.Switch(c, &dark, "Dark theme").Changed() {
					m.SetDarkMode(dark)
				}
			})

			ui.Row(c).FillWidth().Items(ui.Center).Children(func() {
				ui.Text(c, "Press Enter to send").FontSize(13.5)
				ui.Spacer(c)
				ui.Switch(c, &m.EnterToSend, "Press Enter to send")
			})

			ui.Row(c).FillWidth().Items(ui.Center).Children(func() {
				ui.Text(c, "Desktop notifications").FontSize(13.5)
				ui.Spacer(c)
				ui.Switch(c, &m.NotifyEnabled, "Desktop notifications")
			})

			ui.Divider(c)
			ui.Text(c, "SafeStore Encrypted Backups (AES-256-GCM)").FontSize(13).Bold()
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
				ui.Wrap(c, m.BackupStatus).FontSize(12).Muted()
			}
		})
	}, func() {
		if ui.Button(c, "Done").Clicked() {
			m.SettingsOpen = false
		}
	})
}

func (m *Model) newChatModal(c ui.C) {
	if !m.NewChatOpen {
		return
	}
	ui.ModalWith(c, &m.NewChatOpen, "New Encrypted Chat", ui.ModalSpec{Width: 420}, func() {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.TextField(c, &m.NewChatName, "Contact name", "e.g. Sam").FillWidth().AutoFocus()
			ui.TextField(c, &m.NewChatAddr, "Chatmail / Email", "sam@nine.testrun.org").FillWidth()
		})
	}, func() {
		if ui.Button(c, "Cancel").Outline().Clicked() {
			m.NewChatOpen = false
		}
		if ui.Button(c, "Create").Disabled(strings.TrimSpace(m.NewChatName) == "").Clicked() {
			m.CreateDemoChat(m.NewChatName, m.NewChatAddr)
		}
	})
}

func (m *Model) webxdcDrawer(c ui.C) {
	if !m.WebxdcOpen {
		return
	}
	title := m.WebxdcTitle
	if title == "" {
		title = "Webxdc App"
	}
	ui.Drawer(c, &m.WebxdcOpen, title, ui.DrawerSpec{Side: ui.DrawerRight, Size: 380}, func() {
		ui.Column(c).Fill().Gap(12).Children(func() {
			if m.WebView != nil {
				ui.Slot(c).Grow(1).FillWidth().NativeView(m.WebView.ID())
			} else {
				ui.Wrap(c, m.WebxdcInfo).FontSize(13.5).Muted()
				ui.Text(c, "Scheme: webxdc://app/index.html").FontSize(12).Mono()
				ui.Text(c, "Policy: Ephemeral=true, BridgeMainFrameOnly=true").FontSize(12).Mono()
			}
		})
	}, func() {
		if ui.Button(c, "Close").Outline().Clicked() {
			if m.WebView != nil {
				m.WebView.Close()
				m.WebView = nil
			}
			m.WebxdcOpen = false
		}
	})
}

func bubbleBg(theme *gfx.Theme, outgoing bool) gfx.Color {
	if outgoing {
		return theme.Primary
	}
	return theme.Secondary
}

func initialsFrom(name string) string {
	return dc.ChatEntry{Name: name}.Initials()
}
