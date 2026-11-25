package appui

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/preview"
)

const (
	historyDefaultStep   = 30
	longMsgTriggerLines  = 8
	longMsgPreviewLines  = 6
	longMsgTriggerWeight = 480
	longMsgPreviewWeight = 340
	longMsgMinHiddenRune = 24
	longMsgMinHiddenLine = 2
)

type messageFoldInfo struct {
	Collapsible bool
	Preview     string
	TotalLines  int
	TotalRunes  int
}

func runeVisualWeight(r rune) int {
	if r >= 0x1100 {
		return 2
	}
	return 1
}

// collapseMessageText folds overly tall or long messages into a clean preview
// while preserving UTF-8 rune boundaries and preferring word/punctuation breaks.
func collapseMessageText(text string) messageFoldInfo {
	if text == "" {
		return messageFoldInfo{}
	}
	norm := strings.ReplaceAll(text, "\r\n", "\n")
	norm = strings.ReplaceAll(norm, "\r", "\n")

	totalLines := 1
	totalRunes := 0
	totalWeight := 0
	for _, r := range norm {
		totalRunes++
		if r == '\n' {
			totalLines++
			continue
		}
		totalWeight += runeVisualWeight(r)
	}

	info := messageFoldInfo{
		TotalLines: totalLines,
		TotalRunes: totalRunes,
	}
	if totalLines <= longMsgTriggerLines && totalWeight <= longMsgTriggerWeight {
		return info
	}

	lines := 1
	weight := 0
	cut := len(norm)
	cutByWeight := false
	for idx, r := range norm {
		if r == '\n' {
			if lines >= longMsgPreviewLines {
				cut = idx
				break
			}
			lines++
			continue
		}
		w := runeVisualWeight(r)
		if weight+w > longMsgPreviewWeight {
			cut = idx
			cutByWeight = true
			break
		}
		weight += w
	}

	if cutByWeight && cut > 0 && cut < len(norm) {
		minCut := (cut * 65) / 100
		best := -1
		for i := cut; i > minCut; {
			r, size := utf8.DecodeLastRuneInString(norm[:i])
			i -= size
			if unicode.IsSpace(r) {
				best = i
				break
			}
			if strings.ContainsRune("，。；！？、.,;!?)", r) {
				best = i + size
				break
			}
		}
		if best > minCut {
			cut = best
		}
	}

	prefix := strings.TrimRightFunc(norm[:cut], unicode.IsSpace)
	if prefix == "" {
		return info
	}
	previewRunes := utf8.RuneCountInString(prefix)
	previewLines := strings.Count(prefix, "\n") + 1
	hiddenRunes := totalRunes - previewRunes
	hiddenLines := totalLines - previewLines
	if hiddenRunes < longMsgMinHiddenRune && hiddenLines < longMsgMinHiddenLine {
		return info
	}

	info.Collapsible = true
	info.Preview = prefix + "…"
	return info
}

func messageFoldSummary(info messageFoldInfo) string {
	if info.TotalLines > 1 {
		return fmt.Sprintf("%d lines · %d chars", info.TotalLines, info.TotalRunes)
	}
	return fmt.Sprintf("%d chars", info.TotalRunes)
}

func (m *Model) messageEffectivelyExpanded(msg dc.Message) bool {
	if m.IsMessageExpanded(msg.ID) {
		return true
	}
	if m.SearchOpen {
		q := strings.ToLower(strings.TrimSpace(m.SearchQuery))
		if q != "" && strings.Contains(strings.ToLower(msg.Text), q) {
			return true
		}
	}
	return false
}

func (m *Model) renderFoldFooter(c ui.C, msgID int, info messageFoldInfo, expanded bool, inBubble bool) {
	pal := m.Palette
	if inBubble {
		ui.Row(c).FillWidth().Height(1).Background(pal.Border.WithAlpha(0.45))
	}
	toggleText := "Show more"
	toggleIcon := IconChevronDown
	hitLabel := fmt.Sprintf("Expand message %d", msgID)
	if expanded {
		toggleText = "Show less"
		toggleIcon = IconChevronUp
		hitLabel = fmt.Sprintf("Collapse message %d", msgID)
	}
	ui.Row(c).FillWidth().Items(ui.Center).Gap(8).Children(func() {
		ui.Text(c, messageFoldSummary(info)).FontSize(11).Color(pal.SecondaryText).Ellipsis().Grow(1)
		toggleBtn := ui.Row(c.Key("fold")).Height(24).PadX(8).Radius(6).Items(ui.Center).Gap(4).
			Background(pal.Chat.WithAlpha(0.4)).
			Hover(ui.Look{}.Background(pal.Hover)).
			Hit(ui.ID(c), hitLabel)
		toggleBtn.Children(func() {
			ui.Text(c, toggleText).FontSize(11.5).Weight(gfx.WeightMedium).Color(pal.SelectedText)
			ui.Icon(c, toggleIcon).Size(12, 12).Color(pal.SelectedText)
		})
		if toggleBtn.Clicked() {
			id := msgID
			c.After(func() { m.ToggleMessageExpanded(id) })
		}
	})
}

var quickEmojis = []string{"👍", "❤️", "🔥", "🎉", "😄", "✅", "👀", "🚀", "🙏", "💡"}

func (m *Model) conversationPane(c ui.C) {
	pal := m.Palette
	activeChat := m.ActiveChat()

	ui.Row(c).Grow(1).MinWidth(0).FillHeight().Clip().Background(pal.Chat).Children(func() {
		// Main chat column
		ui.Column(c).Grow(1).MinWidth(0).FillHeight().Clip().Background(pal.Chat).Children(func() {
			if activeChat == nil {
				m.emptyChatSurface(c)
				return
			}

			m.chatHeaderBar(c, activeChat)

			if m.Selecting {
				m.multiSelectBar(c, activeChat)
			}
			if pinned := m.PinnedMessages(); len(pinned) > 0 {
				ui.Row(c).FillWidth().PadX(18).PadY(7).Gap(8).Items(ui.Center).
					Background(pal.Header).Children(func() {
					ui.Icon(c, IconPin).Size(13, 13).Color(c.Theme().Primary)
					ui.Text(c, "Pinned").FontSize(11.5).Weight(gfx.WeightSemibold).Color(c.Theme().Primary).Shrink(0)
					ui.Text(c, pinned[0].Text).FontSize(12).Color(pal.SecondaryText).Ellipsis().Grow(1)
				})
				ui.Row(c).FillWidth().Height(1).Background(pal.Border)
			}
			if m.SearchOpen {
				m.chatSearchBar(c)
			}

			// Message timeline
			m.chatTimeline(c, activeChat)

			// GoIM-style docked composer
			m.dockedComposer(c)
		})

		// GoIM-style right docked Chat Files inspector pane
		if activeChat != nil && m.ChatFilesOpen {
			ui.Row(c).Width(1).FillHeight().Background(pal.Border)
			m.chatFilesDockedPane(c, activeChat)
		}
	})
}

func (m *Model) emptyChatSurface(c ui.C) {
	pal := m.Palette
	ui.Column(c).Fill().Center().Pad(36).Gap(12).Children(func() {
		ui.Row(c).Size(62, 62).Radius(18).Background(pal.MutedAvatar).Center().Children(func() {
			ui.Icon(c, IconMessage).Size(28, 28).Color(pal.SecondaryText)
		})
		ui.Text(c, "Select a conversation").FontSize(17).Weight(gfx.WeightSemibold)
		ui.Text(c, "Choose a Delta Chat conversation on the left, or pair via a SecureJoin invite link.").
			FontSize(13).Color(pal.SecondaryText).TextCenter()
		ui.Row(c).Gap(10).Children(func() {
			if ui.Button(c, "Invite / Join Chat").Small().Clicked() {
				m.InviteOpen = true
			}
			if ui.Button(c, "New Encrypted Chat").Small().Outline().Clicked() {
				m.NewChatOpen = true
			}
		})
	})
}

func (m *Model) chatHeaderBar(c ui.C, activeChat *dc.ChatEntry) {
	pal := m.Palette
	theme := c.Theme()

	ui.Column(c).FillWidth().Shrink(0).Background(pal.Header).Children(func() {
		ui.Row(c).FillWidth().Height(ChatHeaderHeight).PadX(18).Gap(11).Items(ui.Center).Children(func() {
			m.renderAvatar(c, activeChat.Name, activeChat.AvatarPath, 38, activeChat.WasSeenRecently, pal.Header)

			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Justify(ui.Center).Children(func() {
				ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
					ui.Text(c, activeChat.Name).FontSize(16).Weight(gfx.WeightSemibold).Color(theme.Foreground).Ellipsis().Shrink(1)
					if activeChat.IsMuted {
						ui.Icon(c, IconBellOff).Size(14, 14).Color(pal.SecondaryText)
					}
				})
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
				dotColor := pal.SecondaryText
				if activeChat.WasSeenRecently {
					dotColor = pal.Success
				}
				ui.Row(c).Gap(6).Items(ui.Center).Children(func() {
					ui.Row(c).Size(6, 6).Radius(3).Background(dotColor).Shrink(0)
					ui.Text(c, meta).FontSize(11.5).Color(pal.SecondaryText).Ellipsis()
				})
			})

			// GoIM-style right header action bar: Find, Chat Files drawer, Mute, More
			ui.Row(c).Shrink(0).Gap(4).Items(ui.Center).Children(func() {
				if m.shellIconBtn(c, "Find", IconSearch, 34, 18, m.SearchOpen).Clicked() {
					m.SearchOpen = !m.SearchOpen
				}
				if m.shellIconBtn(c, "Chat Files", IconFolder, 34, 18, m.ChatFilesOpen).Clicked() {
					m.ChatFilesOpen = !m.ChatFilesOpen
				}
				muteLabel := "Mute"
				muteIcon := IconBell
				if activeChat.IsMuted {
					muteLabel = "Unmute"
					muteIcon = IconBellOff
				}
				if m.shellIconBtn(c, muteLabel, muteIcon, 34, 18, activeChat.IsMuted).Clicked() {
					id := activeChat.ID
					c.After(func() { m.ToggleMuteChat(id) })
				}
				moreBtn := m.shellIconBtn(c, "More", IconMore, 34, 18, m.MoreMenuOpen)
				if moreBtn.Clicked() {
					m.MoreMenuOpen = !m.MoreMenuOpen
				}
				ui.Menu(c, moreBtn, &m.MoreMenuOpen, func(menu *ui.MenuItems) {
					if menu.Item("Chat Info").Clicked() {
						m.ChatInfoOpen = true
					}
					if menu.Item("Media & Files Gallery").Clicked() {
						m.GalleryOpen = true
					}
					muteMenuLabel := "Mute notifications"
					if activeChat.IsMuted {
						muteMenuLabel = "Unmute notifications"
					}
					if menu.Item(muteMenuLabel).Clicked() {
						id := activeChat.ID
						c.After(func() { m.ToggleMuteChat(id) })
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
		})
		ui.Row(c).FillWidth().Height(1).Background(pal.Border)
	})
}

func (m *Model) multiSelectBar(c ui.C, activeChat *dc.ChatEntry) {
	selectedIDs := m.SelectedMessageIDs()
	ui.Row(c).FillWidth().PadX(18).PadY(8).Gap(10).Items(ui.Center).
		Background(m.Palette.Selected).Children(func() {
		ui.Textf(c, "%d selected", len(selectedIDs)).FontSize(12.5).Weight(gfx.WeightSemibold).Color(m.Palette.SelectedText)
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

func (m *Model) chatSearchBar(c ui.C) {
	pal := m.Palette
	matches := m.SearchMatches()
	ui.Row(c).FillWidth().PadX(18).PadY(8).Gap(8).Items(ui.Center).
		Background(pal.Header).Children(func() {
		ui.TextField(c, &m.SearchQuery, "Find in chat", "Search messages...").Icon(IconSearch).Grow(1).Height(34)
		countText := "0 matches"
		if len(matches) > 0 {
			idx := m.SearchMatch % len(matches)
			countText = fmt.Sprintf("%d / %d", idx+1, len(matches))
		}
		ui.Text(c, countText).FontSize(12).Color(pal.SecondaryText).Mono()
		if ui.Button(c, "Next").XSmall().Outline().Disabled(len(matches) == 0).Clicked() {
			m.SearchMatch = (m.SearchMatch + 1) % len(matches)
		}
		if ui.Button(c, "Close").XSmall().Ghost().Clicked() {
			m.SearchOpen = false
			m.SearchQuery = ""
		}
	})
	ui.Row(c).FillWidth().Height(1).Background(pal.Border)
}

func (m *Model) chatTimeline(c ui.C, activeChat *dc.ChatEntry) {
	pal := m.Palette
	msgs := m.Messages[activeChat.ID]

	ui.Scroll(c.KeyInt(activeChat.ID)).Grow(1).FillWidth().MinHeight(0).
		PadSides(12, 20, 16, 20).Gap(12).Follow(true).Scrollbars().Children(func() {
		if win, ok := m.Windows[activeChat.ID]; ok && win.HasEarlier() {
			ui.Row(c).FillWidth().Justify(ui.Center).Children(func() {
				if ui.Button(c, "Load earlier messages").XSmall().Outline().Clicked() {
					win.ExpandEarlier(historyDefaultStep)
					m.Windows[activeChat.ID] = win
				}
			})
		}

		if len(msgs) == 0 {
			ui.Column(c).FillWidth().PadY(48).Items(ui.Center).Gap(8).Children(func() {
				ui.Row(c).Size(50, 50).Radius(15).Background(pal.MutedAvatar).Center().Children(func() {
					ui.Icon(c, IconMessage).Size(22, 22).Color(pal.SecondaryText)
				})
				ui.Text(c, "No messages yet").FontSize(15).Weight(gfx.WeightMedium)
				ui.Text(c, "Send an end-to-end encrypted message to begin.").FontSize(12).Color(pal.SecondaryText)
			})
			return
		}

		lastDate := ""
		for i := range msgs {
			day := msgs[i].DateLabel()
			newDay := day != "" && day != lastDate
			if newDay {
				lastDate = day
				ui.Row(c.Key("day-"+day)).FillWidth().Height(26).Items(ui.Center).Justify(ui.Center).Children(func() {
					ui.Text(c, day).FontSize(11.5).Color(pal.SecondaryText)
				})
			}
			grouped := !newDay && i > 0 && !msgs[i-1].IsInfo && !msgs[i].IsInfo &&
				msgs[i].IsOutgoing == msgs[i-1].IsOutgoing &&
				msgs[i].DisplaySender() == msgs[i-1].DisplaySender()
			m.renderMessage(c, activeChat, msgs[i], grouped)
		}
	})
}

func (m *Model) renderMessage(c ui.C, chat *dc.ChatEntry, msg dc.Message, grouped bool) {
	theme := c.Theme()
	pal := m.Palette

	if msg.IsInfo {
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Justify(ui.Center).PadY(4).Children(func() {
			ui.Row(c).PadSides(5, 12, 5, 12).Radius(8).Background(pal.MutedAvatar).Children(func() {
				ui.Text(c, msg.Text).FontSize(11.5).Color(pal.SecondaryText)
			})
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
		fold := collapseMessageText(msg.Text)
		expanded := m.messageEffectivelyExpanded(msg)
		displayText := msg.Text
		if fold.Collapsible && !expanded {
			displayText = fold.Preview
		}
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Gap(8).Items(ui.Start).Children(func() {
			ui.Text(c, msg.TimeLabel()).FontSize(11.5).Color(pal.SecondaryText).Mono()
			ui.Text(c, "<"+msg.DisplaySender()+">").FontSize(13).Weight(gfx.WeightSemibold).Color(theme.Primary)
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Wrap(c, displayText).FontSize(13.5)
				if fold.Collapsible {
					m.renderFoldFooter(c, msg.ID, fold, expanded, false)
				}
			})
		})
		return
	}

	if m.Layout == LayoutWorkspace {
		fold := collapseMessageText(msg.Text)
		expanded := m.messageEffectivelyExpanded(msg)
		displayText := msg.Text
		if fold.Collapsible && !expanded {
			displayText = fold.Preview
		}
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Pad(8).Gap(10).Radius(8).
			Hover(ui.Look{}.Background(pal.Hover)).Children(func() {
			m.renderAvatar(c, msg.DisplaySender(), "", 28, false, pal.Chat)
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Row(c).Gap(8).Items(ui.Center).Children(func() {
					ui.Text(c, msg.DisplaySender()).FontSize(13.5).Weight(gfx.WeightSemibold)
					ui.Text(c, msg.TimeLabel()).FontSize(11.5).Color(pal.SecondaryText)
					if msg.IsEdited {
						ui.Text(c, "(edited)").FontSize(11).Color(pal.SecondaryText)
					}
					if status := msg.StatusLabel(); status != "" {
						ui.Text(c, status).FontSize(11).Color(pal.SecondaryText)
					}
				})
				ui.Wrap(c, displayText).FontSize(13.5)
				if fold.Collapsible {
					m.renderFoldFooter(c, msg.ID, fold, expanded, false)
				}
				m.reactionPills(c, msg)
			})
		})
		return
	}

	// GoIM-style Bubble layout with clear incoming/outgoing distinction and zero double-nested boxes:
	// - Pure text: avatar bottom-aligned alongside bubble, inline timestamp beside bubble
	// - Images: standalone rounded image directly on chat canvas (+ optional sleek caption below)
	// - File / Webxdc: standalone crisp file card directly on chat canvas (+ optional text bubble above)
	metaTime := msg.TimeLabel()
	if msg.IsEdited {
		metaTime += " · edited"
	}
	statusGlyph := ""
	statusColor := pal.SecondaryText
	if msg.IsOutgoing {
		switch msg.State {
		case dc.MsgStateOutPending, dc.MsgStateOutPreparing:
			statusGlyph = "Sending"
		case dc.MsgStateOutFailed:
			statusGlyph = "Failed"
			statusColor = pal.Danger
		case dc.MsgStateOutMdnRcvd:
			statusGlyph = "✓✓"
			statusColor = pal.SelectedText
		case dc.MsgStateOutDelivered:
			statusGlyph = "✓"
		}
	}

	selfName := "You"
	if acct := m.ActiveAccount(); acct != nil && acct.DisplayName != "" {
		selfName = acct.DisplayName
	}

	bmp, hasBmp := m.MessageBitmap(msg)
	hasFileCard := !hasBmp && (msg.ViewType == "Webxdc" || msg.FileName != "")
	isRichMedia := hasBmp || hasFileCard

	var bubbleHit ui.Box

	renderTimeMeta := func(cc ui.C) {
		if metaTime == "" && statusGlyph == "" {
			return
		}
		ui.Row(cc).Gap(4).Items(ui.Center).Shrink(0).Children(func() {
			if msg.IsOutgoing && statusGlyph != "" {
				ui.Text(cc, statusGlyph).FontSize(10.5).Weight(gfx.WeightMedium).Color(statusColor)
			}
			if metaTime != "" {
				ui.Text(cc, metaTime).FontSize(10.5).Color(pal.SecondaryText)
			}
		})
	}

	renderTextBubble := func(cc ui.C, text string, quote *dc.Quote, maxW float32) ui.Box {
		bg := pal.Incoming
		borderCol := pal.BubbleBorder
		if msg.IsOutgoing {
			bg = pal.Own
			borderCol = pal.OwnBorder
		}
		fold := collapseMessageText(text)
		expanded := m.messageEffectivelyExpanded(msg)
		displayText := text
		if fold.Collapsible && !expanded {
			displayText = fold.Preview
		}
		box := ui.Column(cc).MaxWidth(maxW).PadSides(9, 13, 9, 13).Radius(RadiusControl).
			Background(bg).Border(1, borderCol).Gap(6).
			Hit(ui.ID(cc), fmt.Sprintf("Message %d", msg.ID))
		if fold.Collapsible {
			box.MinWidth(230)
		}
		box.Children(func() {
			if quote != nil {
				ui.Row(cc).Gap(8).Pad(7).Radius(8).Background(pal.Chat.WithAlpha(0.55)).Items(ui.Center).Children(func() {
					ui.Row(cc).Width(3).Height(26).Radius(1.5).Background(theme.Primary).Shrink(0)
					ui.Column(cc).Grow(1).MinWidth(0).Gap(1).Children(func() {
						ui.Text(cc, quote.AuthorName).FontSize(11.5).Weight(gfx.WeightSemibold).Color(pal.SelectedText)
						ui.Text(cc, quote.Text).FontSize(12).Color(pal.SecondaryText).Ellipsis()
					})
				})
			}
			if displayText != "" {
				ui.Wrap(cc, displayText).FontSize(13.5).Color(theme.Foreground).LineHeight(20)
			}
			if fold.Collapsible {
				m.renderFoldFooter(cc, msg.ID, fold, expanded, true)
			}
		})
		return box
	}

	renderMediaOrBubble := func(cc ui.C) {
		// 1. Standalone Image (GoIM style: clean rounded image on canvas, optional caption below)
		if hasBmp {
			w, h := fitImageSize(bmp, 310, 175)
			colAlign := ui.Start
			if msg.IsOutgoing {
				colAlign = ui.End
			}
			ui.Column(cc).Gap(6).Items(colAlign).Children(func() {
				if msg.Quote != nil {
					renderTextBubble(cc, "", msg.Quote, w)
				}
				imgBox := ui.Row(cc).Size(w, h).Radius(RadiusControl).Clip().Background(pal.MutedAvatar).
					Border(1, pal.BubbleBorder).Hit(ui.ID(cc), "Preview "+msg.FileName)
				imgBox.Children(func() {
					ui.Image(cc, bmp).Size(w, h).Radius(RadiusControl)
				})
				if imgBox.Clicked() {
					copied := msg
					cc.After(func() { m.OpenPreviewForMessage(copied) })
				}
				bubbleHit = imgBox
				if msg.Text != "" {
					renderTextBubble(cc, msg.Text, nil, w)
				}
			})
			return
		}

		// 2. Standalone File / Webxdc Card (GoIM style: zero outer bubble wrapping the file card!)
		if hasFileCard {
			colAlign := ui.Start
			if msg.IsOutgoing {
				colAlign = ui.End
			}
			ui.Column(cc).Gap(6).Items(colAlign).Children(func() {
				if msg.Text != "" || msg.Quote != nil {
					renderTextBubble(cc, msg.Text, msg.Quote, 440)
				}
				bubbleHit = m.renderFileCard(cc, msg)
			})
			return
		}

		// 3. Clean Text Bubble
		bubbleHit = renderTextBubble(cc, msg.Text, msg.Quote, 460)
	}

	rowAlign := ui.End
	if isRichMedia {
		rowAlign = ui.Start
	}

	if msg.IsOutgoing {
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Justify(ui.End).Items(rowAlign).Gap(9).Children(func() {
			ui.Column(c).Items(ui.End).Gap(4).Children(func() {
				if isRichMedia {
					renderMediaOrBubble(c)
					renderTimeMeta(c)
				} else {
					ui.Row(c).Items(ui.End).Gap(8).Children(func() {
						renderTimeMeta(c)
						renderMediaOrBubble(c)
					})
				}
				m.reactionPills(c, msg)
			})
			m.renderAvatar(c, selfName, "", 34, false, pal.Chat)
		})
	} else {
		_, senderTint := avatarAccentColor(msg.DisplaySender(), m.DarkMode)
		ui.Row(c.KeyInt(msg.ID)).FillWidth().Justify(ui.Start).Items(rowAlign).Gap(10).Children(func() {
			m.renderAvatar(c, msg.DisplaySender(), "", 34, false, pal.Chat)
			ui.Column(c).Items(ui.Start).Gap(4).Children(func() {
				if !grouped && chat.Kind == dc.ChatGroup {
					ui.Text(c, msg.DisplaySender()).FontSize(11.5).Weight(gfx.WeightMedium).Color(senderTint).Ellipsis()
				}
				if isRichMedia {
					renderMediaOrBubble(c)
					renderTimeMeta(c)
				} else {
					ui.Row(c).Items(ui.End).Gap(8).Children(func() {
						renderMediaOrBubble(c)
						renderTimeMeta(c)
					})
				}
				m.reactionPills(c, msg)
			})
		})
	}

	if bubbleHit.Valid() {
		m.attachMessageContextMenu(c, bubbleHit, msg)
	}
}

func (m *Model) renderFileCard(c ui.C, msg dc.Message) ui.Box {
	pal := m.Palette
	theme := c.Theme()
	_, tag, _, _ := fileBadgeVisual(msg.FileName, msg.ViewType, m.DarkMode)

	meta := tag
	if sz := formatBytesShort(msg.FileBytes); sz != "" {
		meta = sz + " · " + tag
	}
	avail := "Available locally"
	actionText := "Preview"
	btnLabel := "Preview " + msg.FileName
	if msg.ViewType == "Webxdc" {
		avail = "Sandboxed Webxdc Mini-App"
		actionText = "Launch App"
		btnLabel = "Open Webxdc"
	} else if msg.FilePath == "" {
		avail = "Delta Chat attachment"
	}

	card := ui.Column(c).Width(336).Pad(13).Radius(RadiusControl).Gap(9).
		Background(pal.FileCard).Border(1, pal.FileBorder).
		Hover(ui.Look{}.Background(pal.Hover)).
		Hit(ui.ID(c), btnLabel)
	card.Children(func() {
		ui.Row(c).FillWidth().Items(ui.Center).Gap(11).Children(func() {
			renderFileBadge(c, msg.FileName, msg.ViewType, m.DarkMode, 40, 20)
			ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
				ui.Text(c, msg.FileName).FontSize(13.5).Weight(gfx.WeightSemibold).Color(theme.Foreground).Ellipsis()
				ui.Text(c, meta).FontSize(11.5).Color(pal.SecondaryText).Ellipsis()
			})
		})
		ui.Row(c).FillWidth().Height(1).Background(pal.Border.WithAlpha(0.6))
		ui.Row(c).FillWidth().Items(ui.Center).Gap(8).Children(func() {
			ui.Text(c, avail).FontSize(11).Color(pal.Success).Ellipsis().Grow(1)
			ui.Text(c, actionText).FontSize(11.5).Weight(gfx.WeightMedium).Color(pal.SelectedText).Shrink(0)
		})
	})
	if card.Clicked() {
		copied := msg
		c.After(func() {
			if copied.ViewType == "Webxdc" {
				m.OpenWebxdcMessage(copied)
			} else {
				m.OpenPreviewForMessage(copied)
			}
		})
	}
	return card
}

func (m *Model) attachMessageContextMenu(c ui.C, box ui.Box, msg dc.Message) {
	ui.ContextMenu(c, box, func(menu *ui.MenuItems) {
		if msg.Text != "" && menu.Item("Copy Text").Clicked() {
			_ = app.SetClipboardText(msg.Text)
			m.Toaster.Show("Copied", "Message text copied to clipboard", ui.ToastInfo)
		}
		if collapseMessageText(msg.Text).Collapsible {
			foldLabel := "Expand message"
			if m.IsMessageExpanded(msg.ID) {
				foldLabel = "Collapse message"
			}
			if menu.Item(foldLabel).Clicked() {
				id := msg.ID
				c.After(func() { m.ToggleMessageExpanded(id) })
			}
		}
		if msg.FilePath != "" && menu.Item("Copy File").Clicked() {
			_ = app.SetClipboardFiles([]string{msg.FilePath})
			m.Toaster.Show("Copied", msg.FileName+" copied to clipboard", ui.ToastInfo)
		}
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
		if msg.FileName != "" && menu.Item("Preview attachment").Clicked() {
			copied := msg
			c.After(func() { m.OpenPreviewForMessage(copied) })
		}
		pinLabel := "Pin message"
		if msg.IsPinned {
			pinLabel = "Unpin message"
		}
		if menu.Item(pinLabel).Clicked() {
			c.After(func() { m.TogglePinMessage(msg.ChatID, msg.ID) })
		}
		if msg.IsOutgoing && menu.Item("Edit").Clicked() {
			c.After(func() {
				m.EditingMsgID = msg.ID
				m.Draft = msg.Text
			})
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
	pal := m.Palette
	ui.Row(c).Gap(5).Items(ui.Center).Children(func() {
		for _, rx := range msg.Reactions {
			emoji := rx.Emoji
			label := fmt.Sprintf("%s %d", rx.Emoji, rx.Count)
			bg := pal.Incoming
			fg := pal.SecondaryText
			if rx.BySelf {
				bg = pal.Selected
				fg = pal.SelectedText
			}
			pill := ui.Row(c.Key("rx-"+emoji)).Height(24).PadX(8).Radius(12).Center().
				Background(bg).Border(1, pal.BubbleBorder).Hover(ui.Look{}.Background(pal.Hover)).
				Hit(ui.ID(c), "React "+emoji)
			pill.Children(func() {
				ui.Text(c, label).FontSize(11.5).Color(fg)
			})
			if pill.Clicked() {
				c.After(func() { m.ToggleReaction(msg.ChatID, msg.ID, emoji) })
			}
		}
	})
}

func (m *Model) dockedComposer(c ui.C) {
	pal := m.Palette
	theme := c.Theme()

	ui.Column(c).FillWidth().Shrink(0).Background(pal.Chat).Children(func() {
		// Hairline divider separating timeline from docked composer (GoIM style)
		ui.Row(c).FillWidth().Height(1).Background(pal.Border)

		// Editing / Reply banner
		if m.EditingMsgID > 0 {
			ui.Row(c).FillWidth().PadX(16).PadY(6).Gap(10).Items(ui.Center).
				Background(pal.Header).Children(func() {
				ui.Text(c, "Editing message").FontSize(12).Weight(gfx.WeightSemibold).Color(theme.Primary)
				ui.Spacer(c)
				if ui.Button(c, "Cancel edit").XSmall().Ghost().Clicked() {
					m.EditingMsgID = 0
					m.Draft = ""
				}
			})
		} else if m.ReplyTo != nil {
			ui.Row(c).FillWidth().PadX(16).PadY(6).Gap(10).Items(ui.Center).
				Background(pal.Header).Children(func() {
				ui.Row(c).Width(3).Height(18).Radius(1.5).Background(theme.Primary).Shrink(0)
				ui.Text(c, "Replying to "+m.ReplyTo.DisplaySender()+":").FontSize(12).Weight(gfx.WeightSemibold).Color(pal.SelectedText).Shrink(0)
				ui.Text(c, m.ReplyTo.Text).FontSize(12).Color(pal.SecondaryText).Ellipsis().Grow(1)
				if ui.Button(c, "Cancel reply").XSmall().Ghost().Clicked() {
					m.ReplyTo = nil
				}
			})
		}

		// Staged multi-file attachment tray
		if len(m.PendingFiles) > 0 {
			m.stagedAttachmentsTray(c)
		}

		// Compact quick emoji picker bar
		if m.EmojiOpen {
			ui.Row(c).FillWidth().PadX(14).PadY(6).Gap(4).Items(ui.Center).
				Background(pal.Header).Children(func() {
				for _, em := range quickEmojis {
					emoji := em
					btn := ui.Row(c.Key("em-"+emoji)).Size(28, 28).Radius(7).Center().
						Hover(ui.Look{}.Background(pal.Hover)).Hit(ui.ID(c), "Insert "+emoji)
					btn.Children(func() {
						ui.Text(c, emoji).FontSize(15)
					})
					if btn.Clicked() {
						m.Draft += emoji
						m.EmojiOpen = false
					}
				}
			})
			ui.Row(c).FillWidth().Height(1).Background(pal.Border)
		}

		// GoIM-style Composer Toolbar Row
		ui.Row(c).FillWidth().Height(ComposerToolbarHeight).PadX(12).Gap(4).Items(ui.Center).Children(func() {
			if m.shellIconBtn(c, "Emoji", IconSmile, 32, 18, m.EmojiOpen).Clicked() {
				m.EmojiOpen = !m.EmojiOpen
			}
			if m.shellIconBtn(c, "Attach file", IconPaperclip, 32, 18, false).Clicked() {
				opts := app.OpenFileOptions{
					Title:         "Select files to send",
					AllowMultiple: true,
				}
				c.After(func() { m.PickAttachments(opts, []string{"release-notes.md"}) })
			}
			if m.shellIconBtn(c, "Attach photos", IconImage, 32, 18, false).Clicked() {
				opts := app.OpenFileOptions{
					Title:         "Select images to send",
					AllowMultiple: true,
					Filters: []app.FileFilter{
						{Name: "Images", Pattern: "*.png;*.jpg;*.jpeg;*.gif"},
					},
				}
				c.After(func() { m.PickAttachments(opts, []string{"photo-1.png", "photo-2.jpg"}) })
			}
			ui.Spacer(c)
			ui.Row(c).Gap(5).Items(ui.Center).PadX(6).Children(func() {
				ui.Icon(c, IconShield).Size(13, 13).Color(pal.Success)
				ui.Text(c, "Autocrypt E2EE").FontSize(11).Color(pal.SecondaryText)
			})
		})

		// Composer Editor Input (GoIM-proportioned spacious docked typing area)
		ui.Column(c).FillWidth().Height(ComposerEditorHeight).PadSides(2, 16, 4, 16).Justify(ui.Start).Children(func() {
			if m.EnterToSend {
				field := ui.TextField(c, &m.Draft, "Message", "Write a message...").
					FillWidth().Height(40).Background(gfx.Color{}).Border(0, gfx.Color{}).Shadow(gfx.ShadowNone)
				if field.Submitted() {
					m.SendDraft()
				}
			} else {
				ui.TextArea(c, &m.Draft, "Message", "Write a message...").
					FillWidth().Height(ComposerEditorHeight - 6).Background(gfx.Color{}).Border(0, gfx.Color{}).Shadow(gfx.ShadowNone)
			}
		})

		// GoIM-style Composer Footer: keyboard hint on left, primary Send button on right
		hint := "Enter to send · Ctrl+V paste image or files"
		if !m.EnterToSend {
			hint = "Ctrl+Enter to send · Enter for newline"
		}
		sendLabel := "Send"
		sendVisual := "Send"
		if m.EditingMsgID > 0 {
			sendLabel = "Save"
			sendVisual = "Save"
		} else if len(m.PendingFiles) > 1 {
			sendLabel = fmt.Sprintf("Send (%d)", len(m.PendingFiles))
			sendVisual = sendLabel
		}
		canSend := strings.TrimSpace(m.Draft) != "" || len(m.PendingFiles) > 0
		sendBg := pal.Chat.Mix(theme.Foreground, 0.08)
		sendFg := pal.SecondaryText
		if canSend {
			sendBg = theme.Primary
			sendFg = theme.PrimaryForeground
		}

		ui.Row(c).FillWidth().Height(ComposerFooterHeight).PadX(16).Items(ui.Center).Gap(12).Children(func() {
			ui.Text(c, hint).FontSize(11.5).Color(pal.SecondaryText).Ellipsis().Grow(1)
			sendBtn := ui.ButtonBase(c, sendLabel).MinWidth(78).Height(30).PadX(16).Radius(RadiusControl).
				Background(sendBg).Center().Disabled(!canSend).
				Children(func() {
					ui.Text(c, sendVisual).FontSize(12.5).Weight(gfx.WeightSemibold).Color(sendFg)
				})
			if sendBtn.Clicked() {
				m.SendDraft()
			}
		})
	})
}

func (m *Model) stagedAttachmentsTray(c ui.C) {
	pal := m.Palette
	theme := c.Theme()
	summary := fmt.Sprintf("%d file(s) staged", len(m.PendingFiles))
	if total := m.PendingTotalBytes(); total > 0 {
		summary += " · " + formatBytesShort(total)
	}
	ui.Column(c).FillWidth().PadX(16).PadY(8).Gap(6).Background(pal.Header).Children(func() {
		ui.Row(c).FillWidth().Gap(8).Items(ui.Center).Children(func() {
			ui.Icon(c, IconPaperclip).Size(13, 13).Color(theme.Primary)
			ui.Text(c, summary).FontSize(11.5).Weight(gfx.WeightSemibold).Color(theme.Foreground).Shrink(0)
			ui.Text(c, "Click to preview").FontSize(11).Color(pal.SecondaryText).Grow(1).Ellipsis()
			if ui.Button(c, "+ Add").XSmall().Outline().Clicked() {
				opts := app.OpenFileOptions{Title: "Select files to send", AllowMultiple: true}
				c.After(func() { m.PickAttachments(opts, nil) })
			}
			if ui.Button(c, "Clear all").XSmall().Ghost().Clicked() {
				m.PendingFiles = nil
			}
		})
		ui.Scroll(c).FillWidth().MaxHeight(120).Scrollbars().Children(func() {
			ui.Flow(c).FillWidth().Gap(8).LineGap(6).Children(func() {
				for i := range m.PendingFiles {
					idx := i
					att := m.PendingFiles[idx]
					kind := preview.DetectKind(att.Name)
					meta := string(kind)
					if sz := formatBytesShort(att.Size); sz != "" {
						meta += " · " + sz
					}
					ui.Row(c.KeyInt(idx)).MaxWidth(240).Pad(6).Gap(8).Radius(RadiusControl).
						Background(pal.FileCard).Border(1, pal.FileBorder).Items(ui.Center).Children(func() {
						previewBtn := ui.ButtonBase(c, "Preview "+att.Name).Grow(1).MinWidth(0).Radius(8).Ghost().Gap(8).Items(ui.Center).Children(func() {
							if att.IsImage && att.Bitmap.Width > 0 {
								ui.Image(c, att.Bitmap).Size(40, 40).Radius(7)
							} else {
								renderFileBadge(c, att.Name, "", m.DarkMode, 40, 18)
							}
							ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
								ui.Text(c, att.Name).FontSize(12).Weight(gfx.WeightMedium).Ellipsis()
								ui.Text(c, meta).FontSize(10.5).Color(pal.SecondaryText).Ellipsis()
							})
						})
						if previewBtn.Clicked() {
							attCopy := att
							c.After(func() { m.OpenPreviewForPending(attCopy) })
						}
						if ui.ButtonBase(c, "Remove attachment").Size(22, 22).Radius(6).Ghost().Center().Shrink(0).Children(func() {
							ui.Icon(c, IconClose).Size(12, 12).Color(pal.SecondaryText)
						}).Clicked() {
							c.After(func() { m.RemoveAttachment(idx) })
						}
					})
				}
			})
		})
	})
	ui.Row(c).FillWidth().Height(1).Background(pal.Border)
}

// chatFilesDockedPane renders the GoIM-style right-side Chat Files inspector panel.
func (m *Model) chatFilesDockedPane(c ui.C, activeChat *dc.ChatEntry) {
	pal := m.Palette
	theme := c.Theme()

	ui.Column(c).Width(ChatFilesPaneWidth).Shrink(0).FillHeight().Background(pal.Sidebar).Children(func() {
		// Header
		ui.Row(c).FillWidth().Height(ChatHeaderHeight).PadX(18).Items(ui.Center).Gap(10).Children(func() {
			ui.Text(c, "Chat Files").FontSize(15.5).Weight(gfx.WeightSemibold).Color(theme.Foreground).Grow(1)
			if m.shellIconBtn(c, "Close Chat Files", IconClose, 32, 16, false).Clicked() {
				m.ChatFilesOpen = false
			}
		})
		ui.Row(c).FillWidth().Height(1).Background(pal.Border)

		// Filter pills + search box + scrollable list
		ui.Column(c).FillWidth().Grow(1).MinHeight(0).Pad(16).Gap(10).Children(func() {
			ui.Row(c).FillWidth().Gap(6).Items(ui.Center).Children(func() {
				for _, tab := range GalleryTabs {
					t := tab
					active := m.ChatFilesFilter == t
					bg := gfx.Color{}
					fg := pal.SecondaryText
					if active {
						bg = pal.Selected
						fg = pal.SelectedText
					}
					btn := ui.Row(c.Key("cf-tab-"+t)).Height(28).PadX(11).Radius(8).Items(ui.Center).
						Background(bg).Hover(ui.Look{}.Background(pal.Hover)).
						Hit(ui.ID(c), "Chat files filter "+t)
					btn.Children(func() {
						ui.Text(c, t).FontSize(11.5).Weight(gfx.WeightMedium).Color(fg)
					})
					if btn.Clicked() {
						m.ChatFilesFilter = t
					}
				}
			})

			ui.TextField(c, &m.ChatFilesQuery, "Search chat files", "Search by filename...").
				Icon(IconSearch).FillWidth().Height(34).Radius(RadiusControl)

			items := m.SharedFiles(activeChat.ID, m.ChatFilesFilter, m.ChatFilesQuery)
			ui.Scroll(c.Key("chat-files-list")).Grow(1).FillWidth().MinHeight(0).Gap(4).Scrollbars().Children(func() {
				if len(items) == 0 {
					ui.Column(c).FillWidth().PadY(32).Items(ui.Center).Gap(6).Children(func() {
						ui.Text(c, "No shared files").FontSize(13).Weight(gfx.WeightMedium)
						ui.Text(c, "Files, photos, and Webxdc apps sent here appear automatically.").FontSize(11.5).Color(pal.SecondaryText).TextCenter()
					})
					return
				}
				for _, it := range items {
					item := it
					msg := item.Message
					meta := msg.DisplaySender()
					if tm := msg.TimeLabel(); tm != "" {
						meta += " · " + tm
					}
					if sz := formatBytesShort(msg.FileBytes); sz != "" {
						meta += " · " + sz
					}
					statusTxt := "Available locally"
					if msg.ViewType == "Webxdc" {
						statusTxt = "Webxdc Mini-App"
					}
					row := ui.Row(c.KeyInt(30000+msg.ID)).FillWidth().MinHeight(62).Pad(10).Radius(RadiusControl).Gap(10).
						Items(ui.Center).Hover(ui.Look{}.Background(pal.Hover)).
						Hit(ui.ID(c), "File "+msg.FileName)
					row.Children(func() {
						renderFileBadge(c, msg.FileName, msg.ViewType, m.DarkMode, 36, 18)
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Justify(ui.Center).Children(func() {
							ui.Text(c, msg.FileName).FontSize(13).Weight(gfx.WeightMedium).Color(theme.Foreground).Ellipsis()
							ui.Text(c, meta).FontSize(10.5).Color(pal.SecondaryText).Ellipsis()
							ui.Text(c, statusTxt).FontSize(10.5).Color(pal.Success).Ellipsis()
						})
						ui.Icon(c, IconChevronRight).Size(14, 14).Color(pal.SecondaryText)
					})
					if row.Clicked() {
						copied := msg
						c.After(func() { m.OpenPreviewForMessage(copied) })
					}
				}
			})

			ui.Row(c).FillWidth().Height(28).Items(ui.Center).Gap(6).Children(func() {
				ui.Icon(c, IconShield).Size(13, 13).Color(pal.SecondaryText)
				ui.Text(c, "Stored in local encrypted Delta Chat cache").FontSize(10.5).Color(pal.SecondaryText).Ellipsis()
			})
		})
	})
}

func fitImageSize(bmp gfx.Bitmap, maxW, maxH float32) (float32, float32) {
	if bmp.Width <= 0 || bmp.Height <= 0 {
		return 240, 150
	}
	scale := min(maxW/float32(bmp.Width), maxH/float32(bmp.Height))
	if scale > 1 {
		scale = 1
	}
	return max(80, float32(bmp.Width)*scale), max(60, float32(bmp.Height)*scale)
}
