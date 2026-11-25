package appui

import (
	"fmt"
	"strings"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/components"
	"github.com/vivaui/viva/ui"
	uimarkdown "github.com/vivaui/viva/ui/markdown"

	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/preview"
)

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
			docPath := doc.Path
			if ui.Button(c, "Open with System App").Small().Outline().Clicked() {
				c.After(func() { m.OpenSystemPath(docPath) })
			}
			if ui.Button(c, "Reveal in Folder").Small().Ghost().Clicked() {
				c.After(func() { m.RevealSystemPath(docPath) })
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
						m.Section = SectionChats
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
					renderFileBadge(c, it.FileName, it.ViewType, m.DarkMode, 30, 15)
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
	ui.ModalWith(c, &m.SettingsOpen, "Parla Viva Settings", ui.ModalSpec{Width: 500}, func() {
		ui.Column(c).FillWidth().Gap(12).Children(func() {
			ui.Text(c, "Color scheme").FontSize(13).Bold()
			scheme := m.ColorScheme
			if ui.Choice(c, &scheme, ColorSchemes, ColorSchemeLabel).Changed() {
				m.SetColorScheme(scheme)
			}
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
		if ui.Button(c, "Create").Disabled(strings.TrimSpace(m.NewChatName) == "" && strings.TrimSpace(m.NewChatAddr) == "").Clicked() {
			m.CreateNewChat(m.NewChatName, m.NewChatAddr)
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
