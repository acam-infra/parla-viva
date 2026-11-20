package appui

import (
	"fmt"
	"strings"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"
	"github.com/vivaui/viva/ui/message"

	"github.com/acam-infra/parla-viva/internal/dc"
)

// Page renders the Telegram-inspired desktop UI for Parla Viva.
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
				ui.Row(c).FillWidth().PadX(14).PadY(6).Gap(10).Items(ui.Center).
					Background(theme.Muted).Border(1, theme.Border).Children(func() {
					ui.Text(c, "Offline Preview").FontSize(12).Bold().Color(theme.Primary).Shrink(0)
					ui.Text(c, m.StatusText).FontSize(12).Muted().Grow(1).Ellipsis()
					if ui.Button(c, "Invite / Join").XSmall().Outline().Clicked() {
						m.InviteOpen = true
					}
					if ui.Button(c, "Settings").XSmall().Ghost().Clicked() {
						m.SettingsOpen = true
					}
				})
			}

			ui.Row(c).Fill().Grow(1).Children(func() {
				m.sidebarPane(c)
				ui.Splitter(c, &m.Split).Mins(260, 380).SplitterThickness(1)
				m.conversationPane(c)
			})
		})

		// Telegram-style drag-and-drop overlay indicator
		if m.DragOver {
			ui.Column(c).Fill().Center().Background(gfx.Color{Alpha: 0.45}).Children(func() {
				ui.Column(c).Pad(28).Radius(14).Background(theme.Card).Border(2, theme.Primary).Dashed().Center().Gap(6).Children(func() {
					ui.Text(c, "Drop images or files to send").FontSize(18).Bold().Color(theme.Primary)
					ui.Text(c, "Supports multiple images (PNG, JPEG, GIF) and documents").FontSize(13).Muted()
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
	m.lightboxModal(c)
	m.settingsModal(c)
	m.newChatModal(c)
	m.webxdcDrawer(c)
	ui.Toasts(c, &m.Toaster)
}

func (m *Model) sidebarPane(c ui.C) {
	theme := c.Theme()
	ui.Column(c).Width(m.Split.Pos).FillHeight().Pad(12).Gap(10).
		Background(theme.Card).Children(func() {
		ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
			ui.Text(c, "Parla Viva").FontSize(16).Bold().Shrink(0)
			ui.Spacer(c)
			if ui.Button(c, "Invite").XSmall().Outline().Clicked() {
				m.InviteOpen = true
			}
			if ui.Button(c, "New").XSmall().Ghost().Clicked() {
				m.NewChatOpen = true
			}
			if ui.Button(c, "Quick").XSmall().Ghost().Clicked() {
				m.QuickOpen = true
				m.QuickQuery = ""
			}
		})

		ui.TextField(c, &m.SidebarQuery, "Search chats", "Search chats or messages...").FillWidth()
		ui.Choice(c, &m.SidebarFilter, ChatFilters, nil)

		chats := m.FilteredChats()
		ui.Scroll(c).Grow(1).Gap(4).Scrollbars().Children(func() {
			if len(chats) == 0 {
				ui.Column(c).FillWidth().PadY(24).Items(ui.Center).Children(func() {
					ui.Text(c, "No matching chats").FontSize(13).Muted()
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
							ui.Text(c, chat.Initials()).FontSize(11).Bold().Color(avatarFg)
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
						preview := chat.LastMessage
						if chat.SummaryPrefix != "" {
							preview = chat.SummaryPrefix + ": " + preview
						}
						if preview == "" {
							preview = "No messages yet"
						}
						ui.Text(c, preview).FontSize(12).Muted().Ellipsis()
					})
				})
				if row.Clicked() {
					id := chat.ID
					c.After(func() { m.SelectChat(id) })
				}
				// Telegram-style right-click context menu on chat list rows
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
			acctLabel := "No account"
			if acct := m.ActiveAccount(); acct != nil {
				acctLabel = acct.DisplayName
			}
			if ui.Button(c, "Accounts").XSmall().Outline().Clicked() {
				m.AccountsOpen = true
			}
			ui.Text(c, acctLabel).FontSize(11.5).Muted().Ellipsis().Grow(1)
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
				ui.Text(c, "Choose a chat on the left, press Ctrl+K, or pair with Parla via Invite / Join.").FontSize(13).Muted()
				if ui.Button(c, "Invite / Join Chat").Small().Outline().Clicked() {
					m.InviteOpen = true
				}
			})
			return
		}

		// Conversation header bar
		ui.Row(c).FillWidth().PadX(18).PadY(12).Gap(8).Items(ui.Center).
			Background(theme.Card).Border(1, theme.Border).Children(func() {
			ui.Column(c).Grow(1).Gap(2).Children(func() {
				ui.Text(c, activeChat.Name).FontSize(15).Bold()
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
			if ui.Button(c, "Gallery").XSmall().Outline().Clicked() {
				m.GalleryOpen = true
			}
			if ui.Button(c, "Info").XSmall().Outline().Clicked() {
				m.ChatInfoOpen = true
			}
			if ui.Button(c, "Select").XSmall().Ghost().Clicked() {
				m.Selecting = !m.Selecting
				if !m.Selecting {
					m.SelectedMsgs = make(map[int]bool)
				}
			}
			if ui.Button(c, "Find").XSmall().Outline().Clicked() {
				m.SearchOpen = !m.SearchOpen
			}
			if ui.Button(c, "Settings").XSmall().Ghost().Clicked() {
				m.SettingsOpen = true
			}
		})

		if m.Selecting {
			selectedIDs := m.SelectedMessageIDs()
			ui.Row(c).FillWidth().PadX(18).PadY(8).Gap(10).Items(ui.Center).
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
			ui.Row(c).FillWidth().PadX(18).PadY(6).Gap(8).Items(ui.Center).
				Background(theme.Muted).Children(func() {
				ui.Text(c, "Pinned").FontSize(11.5).Bold().Color(theme.Primary).Shrink(0)
				ui.Text(c, pinned[0].Text).FontSize(12).Muted().Ellipsis().Grow(1)
			})
		}

		if m.SearchOpen {
			matches := m.SearchMatches()
			ui.Row(c).FillWidth().PadX(18).PadY(8).Gap(8).Items(ui.Center).
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

		// Message transcript
		msgs := m.Messages[activeChat.ID]
		ui.Scroll(c).Grow(1).FillWidth().Pad(18).Gap(10).Follow(true).Scrollbars().Children(func() {
			if win, ok := m.Windows[activeChat.ID]; ok && win.HasEarlier() {
				ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
					if ui.Button(c, "Load earlier messages").XSmall().Outline().Clicked() {
						win.ExpandEarlier(historyDefaultStep)
						m.Windows[activeChat.ID] = win
					}
				})
			}

			for i := range msgs {
				m.renderMessage(c, msgs[i])
			}
		})

		// Editing / Reply banner
		if m.EditingMsgID > 0 {
			ui.Row(c).FillWidth().PadX(18).PadY(6).Gap(10).Items(ui.Center).
				Background(theme.Muted).Children(func() {
				ui.Text(c, "Editing message").FontSize(12).Bold().Color(theme.Primary)
				ui.Spacer(c)
				if ui.Button(c, "Cancel edit").XSmall().Ghost().Clicked() {
					m.EditingMsgID = 0
					m.Draft = ""
				}
			})
		} else if m.ReplyTo != nil {
			ui.Row(c).FillWidth().PadX(18).PadY(6).Gap(10).Items(ui.Center).
				Background(theme.Muted).Children(func() {
				ui.Text(c, "Replying to "+m.ReplyTo.DisplaySender()+":").FontSize(12).Bold().Color(theme.Primary).Shrink(0)
				ui.Text(c, m.ReplyTo.Text).FontSize(12).Muted().Ellipsis().Grow(1)
				if ui.Button(c, "Cancel reply").XSmall().Ghost().Clicked() {
					m.ReplyTo = nil
				}
			})
		}

		// Telegram-style horizontal multi-image / multi-file attachment strip
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
							}
							ui.Text(c, att.Name).FontSize(12).Mono()
							if ui.Button(c, "x").XSmall().Ghost().Clicked() {
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

		// Compose bar with multi-image picker
		ui.Row(c).FillWidth().Pad(12).Gap(8).Items(ui.Center).
			Background(theme.Card).Border(1, theme.Border).Children(func() {
			if ui.Button(c, "+ Photos").Small().Outline().Clicked() {
				opts := app.OpenFileOptions{
					Title:         "Select images or files",
					AllowMultiple: true,
				}
				if paths, err := app.OpenFiles(opts); err == nil && len(paths) > 0 {
					m.AddAttachments(paths)
				} else {
					m.AddAttachments([]string{"photo-1.png", "photo-2.jpg"})
				}
			}
			field := ui.TextField(c, &m.Draft, "Message", "Write a message or drop images...")
			field.Grow(1)
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
			if ui.Button(c, sendLabel).Disabled(!canSend).Clicked() {
				m.SendDraft()
			}
		})
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
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Pad(8).Gap(10).Radius(6).
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
				ui.Wrap(c, msg.Text).FontSize(14)
				m.messageExtras(c, msg)
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
			ui.Column(cc).Gap(6).Pad(10).Radius(10).MaxWidthPercent(75).
				Background(bubbleBg(theme, msg.IsOutgoing)).Children(func() {
				if msg.Quote != nil {
					ui.Column(cc).PadLeft(8).Border(1, theme.Border).Children(func() {
						ui.Text(cc, msg.Quote.AuthorName).FontSize(11.5).Bold()
						ui.Text(cc, msg.Quote.Text).FontSize(12).Muted().Ellipsis()
					})
				}
				if hasBmp {
					imgBox := ui.Row(cc).Size(240, 150).Radius(8).Clip().
						Hit(ui.ID(cc), "View "+msg.FileName)
					imgBox.Children(func() {
						ui.Image(cc, bmp).Size(240, 150).Radius(8)
					})
					if imgBox.Clicked() {
						title := msg.FileName
						bCopy := bmp
						cc.After(func() { m.OpenLightbox(title, bCopy) })
					}
				}
				fg := theme.SecondaryForeground
				if msg.IsOutgoing {
					fg = theme.PrimaryForeground
				}
				if msg.Text != "" {
					ui.Wrap(cc, msg.Text).FontSize(14).Color(fg)
				}
				if msg.ViewType == "Webxdc" {
					ui.Row(cc).Gap(8).Items(ui.Center).Children(func() {
						ui.Text(cc, "Webxdc: "+msg.FileName).FontSize(12).Bold().Color(fg)
						if ui.Button(cc, "Open Webxdc").XSmall().Outline().Clicked() {
							copied := msg
							cc.After(func() { m.OpenWebxdcMessage(copied) })
						}
					})
				} else if msg.FileName != "" && !hasBmp {
					ui.Row(cc).Gap(8).Pad(6).Radius(6).Background(theme.Card).Items(ui.Center).Children(func() {
						ui.Text(cc, "File: "+msg.FileName).FontSize(12).Mono()
						if msg.FileBytes > 0 {
							ui.Textf(cc, "(%d KB)", max(msg.FileBytes/1024, 1)).FontSize(11).Muted()
						}
					})
				}
			})
		})
	} else {
		row = row.Body(msg.Text)
	}

	row = row.Actions(func(ac ui.C) {
		m.messageExtras(ac, msg)
	})

	box := message.View(c.KeyInt(msg.ID), msg.DisplaySender(), row)
	// Telegram-style right-click context menu on every message
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

func (m *Model) messageExtras(c ui.C, msg dc.Message) {
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
		if ui.Button(c, "Reply").XSmall().Ghost().Clicked() {
			copied := msg
			c.After(func() { m.ReplyTo = &copied })
		}
		if ui.Button(c, "+1").XSmall().Ghost().Clicked() {
			c.After(func() { m.ToggleReaction(msg.ChatID, msg.ID, "👍") })
		}
		pinLabel := "Pin"
		if msg.IsPinned {
			pinLabel = "Unpin"
		}
		if ui.Button(c, pinLabel).XSmall().Ghost().Clicked() {
			c.After(func() { m.TogglePinMessage(msg.ChatID, msg.ID) })
		}
		if msg.IsOutgoing {
			if ui.Button(c, "Edit").XSmall().Ghost().Clicked() {
				c.After(func() {
					m.EditingMsgID = msg.ID
					m.Draft = msg.Text
				})
			}
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

func (m *Model) lightboxModal(c ui.C) {
	if !m.LightboxOpen {
		return
	}
	title := m.LightboxTitle
	if title == "" {
		title = "Image Preview"
	}
	ui.ModalWith(c, &m.LightboxOpen, title, ui.ModalSpec{Width: 720}, func() {
		ui.Column(c).FillWidth().Gap(10).Children(func() {
			ui.Slot(c).FillWidth().Height(420).Draw(func(d *ui.DrawContext) {
				components.PaintImageView(d.Canvas, *d.Theme, d.Rect, components.ImageViewOptions{
					Bitmap: m.LightboxBitmap,
					Zoom:   m.LightboxZoom,
				})
			})
			ui.Row(c).FillWidth().Gap(8).Justify(ui.Center).Children(func() {
				if ui.Button(c, "Fit").XSmall().Outline().Clicked() {
					m.LightboxZoom = 0
				}
				if ui.Button(c, "100%").XSmall().Outline().Clicked() {
					m.LightboxZoom = 1
				}
				if ui.Button(c, "Zoom +").XSmall().Outline().Clicked() {
					if m.LightboxZoom == 0 {
						m.LightboxZoom = 1.5
					} else {
						m.LightboxZoom = min(m.LightboxZoom+0.5, 4)
					}
				}
			})
		})
	}, func() {
		if ui.Button(c, "Close").Clicked() {
			m.LightboxOpen = false
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
				ui.Row(c.KeyInt(it.ID)).FillWidth().Pad(8).Gap(10).Radius(6).
					Background(c.Theme().Muted).Items(ui.Center).Children(func() {
					ui.Text(c, it.ViewType).FontSize(11.5).Bold().Color(c.Theme().Primary)
					ui.Text(c, it.FileName).FontSize(13).Mono().Grow(1).Ellipsis()
					if bmp, ok := m.MessageBitmap(it); ok {
						if ui.Button(c, "View").XSmall().Outline().Clicked() {
							title := it.FileName
							bCopy := bmp
							c.After(func() {
								m.GalleryOpen = false
								m.OpenLightbox(title, bCopy)
							})
						}
					} else if it.ViewType == "Webxdc" {
						if ui.Button(c, "Launch").XSmall().Outline().Clicked() {
							copied := it
							c.After(func() {
								m.GalleryOpen = false
								m.OpenWebxdcMessage(copied)
							})
						}
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
