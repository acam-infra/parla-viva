package appui

import (
	"fmt"
	"strings"

	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"
)

// filesWorkspace renders the GoIM-style full-page Files & Media table across all conversations.
func (m *Model) filesWorkspace(c ui.C) {
	pal := m.Palette
	theme := c.Theme()

	allCount := len(m.SharedFiles(0, "All", ""))
	imgCount := len(m.SharedFiles(0, "Images", ""))
	fileCount := len(m.SharedFiles(0, "Files", ""))
	xdcCount := len(m.SharedFiles(0, "Webxdc", ""))

	items := m.SharedFiles(0, m.FilesFilter, m.FilesQuery)

	ui.Column(c).Fill().Background(pal.Chat).PadSides(26, 30, 18, 30).Gap(12).Children(func() {
		// Page header (mirrors GoIM Transfers header)
		ui.Row(c).FillWidth().Items(ui.Center).Gap(12).Children(func() {
			ui.Column(c).Grow(1).MinWidth(0).Gap(4).Children(func() {
				ui.Text(c, "Files & Media").FontSize(24).Weight(gfx.WeightSemibold).Color(theme.Foreground)
				ui.Text(c, "Browse, filter, and preview encrypted attachments, images, and Webxdc mini-apps across chats.").
					FontSize(12.5).Color(pal.SecondaryText).Ellipsis()
			})
			if m.AccountsPath != "" {
				dirBtn := ui.Row(c).Height(38).PadX(14).Gap(8).Radius(8).Items(ui.Center).Shrink(0).
					Border(1, pal.Border).Hover(ui.Look{}.Background(pal.Hover)).
					Hit(ui.ID(c), "Open Data Folder")
				dirBtn.Children(func() {
					ui.Icon(c, IconFolder).Size(16, 16).Color(pal.SecondaryText)
					ui.Text(c, "Data Folder").FontSize(13).Weight(gfx.WeightMedium).Color(theme.Foreground)
				})
				if dirBtn.Clicked() {
					p := m.AccountsPath
					c.After(func() { m.RevealSystemPath(p) })
				}
			}
		})

		// Search box
		ui.Row(c).FillWidth().Children(func() {
			ui.TextField(c, &m.FilesQuery, "Search files", "Search files...").
				Icon(IconSearch).Width(240).Height(34).Radius(RadiusControl)
		})

		// Filter tab bar with live counts (GoIM line-tab style)
		tabs := []struct {
			key   string
			label string
		}{
			{"All", fmt.Sprintf("All %d", allCount)},
			{"Images", fmt.Sprintf("Images %d", imgCount)},
			{"Files", fmt.Sprintf("Documents %d", fileCount)},
			{"Webxdc", fmt.Sprintf("Webxdc %d", xdcCount)},
		}
		ui.Column(c).FillWidth().Gap(0).Children(func() {
			ui.Row(c).FillWidth().Gap(18).Items(ui.Center).Children(func() {
				for _, tab := range tabs {
					t := tab
					active := m.FilesFilter == t.key
					fg := pal.SecondaryText
					if active {
						fg = theme.Foreground
					}
					btn := ui.Column(c.Key("ftab-"+t.key)).PadSides(4, 2, 0, 2).Gap(7).Items(ui.Center).
						Hit(ui.ID(c), "Files tab "+t.key)
					btn.Children(func() {
						ui.Text(c, t.label).FontSize(13.5).Weight(gfx.WeightMedium).Color(fg)
						barCol := gfx.Color{}
						if active {
							barCol = theme.Primary
						}
						ui.Row(c).FillWidth().MinWidth(52).Height(2.5).Radius(1.25).Background(barCol)
					})
					if btn.Clicked() {
						m.FilesFilter = t.key
					}
				}
			})
			ui.Row(c).FillWidth().Height(1).Background(pal.Border)
		})

		// Table header row
		ui.Row(c).FillWidth().Height(30).PadX(8).Gap(12).Items(ui.Center).Children(func() {
			ui.Text(c, "File Name").FontSize(11.5).Color(pal.SecondaryText).Grow(1)
			ui.Text(c, "Conversation").FontSize(11.5).Color(pal.SecondaryText).Width(160).Shrink(0)
			ui.Text(c, "Sender").FontSize(11.5).Color(pal.SecondaryText).Width(110).Shrink(0)
			ui.Text(c, "Size").FontSize(11.5).Color(pal.SecondaryText).Width(84).Shrink(0)
			ui.Text(c, "Status").FontSize(11.5).Color(pal.SecondaryText).Width(120).Shrink(0)
			ui.Text(c, "Action").FontSize(11.5).Color(pal.SecondaryText).Width(92).Shrink(0)
		})

		// Scrollable table body
		ui.Scroll(c.Key("files-table-"+m.FilesFilter)).Grow(1).FillWidth().MinHeight(0).Scrollbars().Children(func() {
			if len(items) == 0 {
				ui.Column(c).FillWidth().PadY(48).Items(ui.Center).Gap(8).Children(func() {
					ui.Row(c).Size(52, 52).Radius(14).Background(pal.MutedAvatar).Center().Children(func() {
						ui.Icon(c, IconFolder).Size(24, 24).Color(pal.SecondaryText)
					})
					ui.Text(c, "No matching files").FontSize(15).Weight(gfx.WeightMedium)
					ui.Text(c, "Try another filter or share files inside any conversation.").FontSize(12).Color(pal.SecondaryText)
				})
				return
			}

			for _, it := range items {
				item := it
				msg := item.Message
				sizeTxt := formatBytesShort(msg.FileBytes)
				if sizeTxt == "" {
					sizeTxt = "—"
				}
				statusTxt := "Available"
				if msg.ViewType == "Webxdc" {
					statusTxt = "Mini-App"
				} else if msg.FilePath == "" {
					statusTxt = "Metadata"
				}

				ui.Column(c.KeyInt(50000+msg.ID)).FillWidth().Children(func() {
					ui.Row(c).FillWidth().Height(54).PadX(8).Gap(12).Radius(8).Items(ui.Center).
						Hover(ui.Look{}.Background(pal.Hover)).Children(func() {
						// File icon + name
						ui.Row(c).Grow(1).MinWidth(0).Gap(10).Items(ui.Center).Children(func() {
							renderFileBadge(c, msg.FileName, msg.ViewType, m.DarkMode, 30, 15)
							ui.Text(c, msg.FileName).FontSize(13.5).Weight(gfx.WeightMedium).Color(theme.Foreground).Ellipsis().Grow(1)
						})

						// Conversation
						chatLabel := item.ChatName
						if chatLabel == "" {
							chatLabel = fmt.Sprintf("Chat #%d", item.ChatID)
						}
						ui.Text(c, chatLabel).FontSize(12.5).Color(pal.SecondaryText).Width(160).Shrink(0).Ellipsis()

						// Sender
						ui.Text(c, msg.DisplaySender()).FontSize(12.5).Color(pal.SecondaryText).Width(110).Shrink(0).Ellipsis()

						// Size
						ui.Text(c, sizeTxt).FontSize(12.5).Color(pal.SecondaryText).Width(84).Shrink(0)

						// Status
						ui.Text(c, statusTxt).FontSize(12.5).Color(pal.Success).Width(120).Shrink(0)

						// Action button
						ui.Row(c).Width(92).Shrink(0).Gap(6).Items(ui.Center).Children(func() {
							actLabel := "Preview"
							if msg.ViewType == "Webxdc" {
								actLabel = "Launch"
							}
							if ui.Button(c.Key("open-"+strings.ToLower(msg.FileName)), actLabel).XSmall().Outline().Clicked() {
								copied := msg
								c.After(func() {
									if copied.ViewType == "Webxdc" {
										m.OpenWebxdcMessage(copied)
									} else {
										m.OpenPreviewForMessage(copied)
									}
								})
							}
						})
					})
					ui.Row(c).FillWidth().Height(1).Background(pal.Border.WithAlpha(0.5))
				})
			}
		})
	})
}
