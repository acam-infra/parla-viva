package appui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vivaui/viva/app"
	"github.com/vivaui/viva/gfx"
	"github.com/vivaui/viva/ui"

	"github.com/acam-infra/parla-viva/internal/backup"
	"github.com/acam-infra/parla-viva/internal/dc"
	"github.com/acam-infra/parla-viva/internal/webxdc"
)

// SwitchAccount changes the active account profile.
func (m *Model) SwitchAccount(acctID int) {
	if acctID == m.ActiveAcctID {
		return
	}
	m.ActiveAcctID = acctID
	m.AccountsOpen = false
	if acct := m.ActiveAccount(); acct != nil {
		m.Toaster.Show("Switched profile", acct.DisplayName, ui.ToastSuccess)
	}
	if !m.DemoMode && m.Client != nil {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := m.Client.SelectAccount(ctx, acctID); err != nil {
				return
			}
			ids, _ := m.Client.GetChatlistEntries(ctx, acctID, nil, "")
			chats, _ := m.Client.GetChatlistItems(ctx, acctID, ids)
			qr, _ := m.Client.GetChatSecurejoinQRCode(ctx, acctID, 0)
			m.EnqueueUpdate(func(model *Model) {
				model.Chats = chats
				model.InviteLink = qr
				model.ActiveChatID = 0
				if len(chats) > 0 {
					model.SelectChat(chats[0].ID)
				}
			})
		}()
	}
}

// CreateNewChatmailAccount provisions a new Chatmail profile (live RPC or demo).
func (m *Model) CreateNewChatmailAccount() {
	name := strings.TrimSpace(m.NewAcctName)
	if name == "" {
		name = "Parla Viva User"
	}
	relay := strings.TrimSpace(m.NewAcctRelay)
	if relay == "" {
		relay = dc.DefaultChatmailRelay
	}
	if m.DemoMode || m.Client == nil {
		newID := len(m.Accounts) + 1
		addr := fmt.Sprintf("%s@nine.testrun.org", strings.ToLower(strings.ReplaceAll(name, " ", "")))
		m.Accounts = append(m.Accounts, dc.Account{ID: newID, Kind: "Configured", DisplayName: name, Addr: addr})
		m.ActiveAcctID = newID
		m.InviteLink = "https://i.delta.chat/#demo-" + addr
		m.OnboardStatus = "Created demo profile: " + addr
		m.Toaster.Show("Account created", addr, ui.ToastSuccess)
		return
	}
	m.OnboardStatus = "Provisioning Chatmail account on " + relay + "..."
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		id, qr, err := m.Client.CreateChatmailAccount(ctx, name, relay)
		if err != nil {
			m.EnqueueUpdate(func(model *Model) {
				model.OnboardStatus = "Account creation failed: " + err.Error()
			})
			return
		}
		accts, _ := m.Client.GetAllAccounts(ctx)
		m.EnqueueUpdate(func(model *Model) {
			model.Accounts = accts
			model.ActiveAcctID = id
			model.InviteLink = qr
			model.OnboardStatus = "Account ready! Share your invite link below."
			model.Toaster.Show("Chatmail ready", name, ui.ToastSuccess)
		})
	}()
}

// JoinViaInviteLink redeems a Delta Chat / Parla invite link (https://i.delta.chat/#... or openpgp4fpr:...).
func (m *Model) JoinViaInviteLink() {
	link := strings.TrimSpace(m.JoinLinkInput)
	if link == "" {
		m.OnboardStatus = "Paste an invite link (https://i.delta.chat/#... or openpgp4fpr:...)."
		return
	}
	if m.DemoMode || m.Client == nil {
		m.CreateDemoChat("Paired Parla Contact", link)
		m.JoinLinkInput = ""
		m.InviteOpen = false
		return
	}
	acctID := m.ActiveAcctID
	m.OnboardStatus = "Joining encrypted chat..."
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		chatID, err := m.Client.SecureJoin(ctx, acctID, link)
		if err != nil {
			m.EnqueueUpdate(func(model *Model) {
				model.OnboardStatus = "Join failed: " + err.Error()
			})
			return
		}
		ids, _ := m.Client.GetChatlistEntries(ctx, acctID, nil, "")
		chats, _ := m.Client.GetChatlistItems(ctx, acctID, ids)
		m.EnqueueUpdate(func(model *Model) {
			model.Chats = chats
			model.JoinLinkInput = ""
			model.InviteOpen = false
			model.SelectChat(chatID)
			model.Toaster.Show("Encrypted chat paired", "SecureJoin started", ui.ToastSuccess)
		})
	}()
}

// CopyInviteLink copies the active account's invite link to the OS clipboard.
func (m *Model) CopyInviteLink() {
	if m.InviteLink == "" {
		return
	}
	_ = app.SetClipboardText(m.InviteLink)
	m.Toaster.Show("Invite link copied", "Paste into Parla on Windows/Linux/macOS", ui.ToastSuccess)
}

// OpenLightbox opens the Telegram-style image preview modal.
func (m *Model) OpenLightbox(title string, bmp gfx.Bitmap) {
	if bmp.Width <= 0 || bmp.Height <= 0 {
		return
	}
	m.LightboxTitle = title
	m.LightboxBitmap = bmp
	m.LightboxZoom = 0 // 0 = fit to viewport
	m.LightboxOpen = true
}

// TogglePinChat pins or unpins the active chat.
func (m *Model) TogglePinChat(chatID int) {
	var pinned bool
	for i := range m.Chats {
		if m.Chats[i].ID == chatID {
			m.Chats[i].IsPinned = !m.Chats[i].IsPinned
			pinned = m.Chats[i].IsPinned
			break
		}
	}
	if !m.DemoMode && m.Client != nil {
		vis := "Normal"
		if pinned {
			vis = "Pinned"
		}
		acctID := m.ActiveAcctID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.Client.SetChatVisibility(ctx, acctID, chatID, vis)
		}()
	}
}

// ToggleMuteChat mutes or unmutes the active chat.
func (m *Model) ToggleMuteChat(chatID int) {
	var muted bool
	for i := range m.Chats {
		if m.Chats[i].ID == chatID {
			m.Chats[i].IsMuted = !m.Chats[i].IsMuted
			muted = m.Chats[i].IsMuted
			break
		}
	}
	if !m.DemoMode && m.Client != nil {
		acctID := m.ActiveAcctID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.Client.SetChatMuteDuration(ctx, acctID, chatID, muted)
		}()
	}
}

// TogglePinMessage pins or unpins a specific message inside a chat.
func (m *Model) TogglePinMessage(chatID, msgID int) {
	msgs := m.Messages[chatID]
	for i := range msgs {
		if msgs[i].ID == msgID {
			msgs[i].IsPinned = !msgs[i].IsPinned
			break
		}
	}
	m.Messages[chatID] = msgs
}

// ApplyEditMessage updates an outgoing message's text locally and via RPC.
func (m *Model) ApplyEditMessage(chatID, msgID int, newText string) {
	msgs := m.Messages[chatID]
	for i := range msgs {
		if msgs[i].ID == msgID {
			msgs[i].Text = newText
			msgs[i].IsEdited = true
			break
		}
	}
	m.Messages[chatID] = msgs
	if !m.DemoMode && m.Client != nil {
		acctID := m.ActiveAcctID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.Client.SendEditRequest(ctx, acctID, msgID, newText)
		}()
	}
}

// DeleteMessageList removes the given message IDs from the active chat.
func (m *Model) DeleteMessageList(chatID int, ids []int) {
	if len(ids) == 0 {
		return
	}
	drop := make(map[int]bool, len(ids))
	for _, id := range ids {
		drop[id] = true
	}
	old := m.Messages[chatID]
	kept := make([]dc.Message, 0, len(old))
	for _, msg := range old {
		if !drop[msg.ID] {
			kept = append(kept, msg)
		}
	}
	m.Messages[chatID] = kept
	m.Selecting = false
	m.SelectedMsgs = make(map[int]bool)

	if !m.DemoMode && m.Client != nil {
		acctID := m.ActiveAcctID
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = m.Client.DeleteMessages(ctx, acctID, ids)
		}()
	}
}

// SelectedMessageIDs returns the currently selected message IDs in multi-select mode.
func (m *Model) SelectedMessageIDs() []int {
	var ids []int
	for id, ok := range m.SelectedMsgs {
		if ok {
			ids = append(ids, id)
		}
	}
	return ids
}

// GalleryItems returns attachments/media in the active chat filtered by GalleryTab.
func (m *Model) GalleryItems() []dc.Message {
	var out []dc.Message
	for _, msg := range m.Messages[m.ActiveChatID] {
		if msg.FileName == "" && msg.ViewType != "Webxdc" && msg.ViewType != "File" && msg.ViewType != "Image" {
			continue
		}
		switch m.GalleryTab {
		case "Images":
			if msg.ViewType != "Image" && !IsImagePath(msg.FileName) {
				continue
			}
		case "Files":
			if msg.ViewType != "File" {
				continue
			}
		case "Webxdc":
			if msg.ViewType != "Webxdc" {
				continue
			}
		}
		out = append(out, msg)
	}
	return out
}

// OpenWebxdcMessage opens a Webxdc mini-app in the drawer (and live WebHost when available).
func (m *Model) OpenWebxdcMessage(msg dc.Message) {
	m.WebxdcTitle = msg.FileName
	if m.WebxdcTitle == "" {
		m.WebxdcTitle = "Webxdc App"
	}
	m.WebxdcInfo = "Offline sandboxed mini-app served via webxdc://app/ with connect-src 'none' CSP and Ephemeral WebView store."
	m.WebxdcOpen = true

	if m.WebHost != nil && m.WebReady {
		if m.WebView != nil {
			m.WebView.Close()
			m.WebView = nil
		}
		if arc, err := webxdc.DemoChecklistArchive(); err == nil {
			cfg := arc.WebViewConfig(func(payload string) {
				m.Toaster.Show("Webxdc update", payload, ui.ToastInfo)
			})
			if view, err := m.WebHost.Open(cfg); err == nil {
				m.WebView = view
			}
		}
	}
}

// CreateEncryptedBackup creates a SafeStore AES-256-GCM encrypted backup of the accounts store.
func (m *Model) CreateEncryptedBackup() {
	if m.BackupPassword == "" {
		m.BackupStatus = "Enter a backup password first."
		return
	}
	src := m.AccountsPath
	if src == "" {
		tmp, err := os.MkdirTemp("", "parla-viva-demo-store-*")
		if err != nil {
			m.BackupStatus = err.Error()
			return
		}
		defer os.RemoveAll(tmp)
		_ = os.WriteFile(filepath.Join(tmp, "profile.json"), []byte(`{"account":"Alice","demo":true}`), 0o600)
		src = tmp
	}
	dst := filepath.Join(os.TempDir(), fmt.Sprintf("parla-viva-backup-%d.pvz", time.Now().UnixNano()))
	info, err := backup.Create(src, dst, m.BackupPassword)
	if err != nil {
		m.BackupStatus = "Backup failed: " + err.Error()
		return
	}
	m.LastBackupPath = dst
	m.BackupStatus = fmt.Sprintf("Encrypted backup saved (%d files, %d bytes)", info.FileCount, info.TotalBytes)
	m.Toaster.Show("Backup created", filepath.Base(dst), ui.ToastSuccess)
}

// VerifyLastBackup verifies the most recently created encrypted backup archive.
func (m *Model) VerifyLastBackup() {
	if m.LastBackupPath == "" {
		m.BackupStatus = "Create a backup first before verifying."
		return
	}
	info, err := backup.Verify(m.LastBackupPath, m.BackupPassword)
	if err != nil {
		m.BackupStatus = "Verification failed: " + err.Error()
		return
	}
	m.BackupStatus = fmt.Sprintf("Verified OK: %d files (%d bytes), SHA-256 intact", info.FileCount, info.TotalBytes)
	m.Toaster.Show("Backup verified", "All SHA-256 hashes match", ui.ToastSuccess)
}
