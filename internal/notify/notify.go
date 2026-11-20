package notify

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Item represents a desktop notification to deliver asynchronously.
type Item struct {
	ChatID int
	Title  string
	Body   string
}

// Notifier delivers native OS desktop notifications asynchronously with
// per-chat throttling so message bursts never stall sync or spam the user.
type Notifier struct {
	ch       chan Item
	mu       sync.Mutex
	lastSent map[int]time.Time
	minGap   time.Duration
	sendFn   func(title, body string) // injectable for tests
	once     sync.Once
}

// New creates and starts a background Notifier with a 1.5s per-chat throttle.
func New() *Notifier {
	n := &Notifier{
		ch:       make(chan Item, 16),
		lastSent: make(map[int]time.Time),
		minGap:   1500 * time.Millisecond,
		sendFn:   deliverNative,
	}
	go n.loop()
	return n
}

// Push enqueues a notification without ever blocking the caller.
func (n *Notifier) Push(item Item) {
	if n == nil || strings.TrimSpace(item.Title) == "" {
		return
	}
	item.Title = truncateClean(item.Title, 64)
	item.Body = truncateClean(item.Body, 140)
	if item.Body == "" {
		item.Body = "New encrypted message"
	}

	n.mu.Lock()
	now := time.Now()
	if item.ChatID > 0 {
		if prev, ok := n.lastSent[item.ChatID]; ok && now.Sub(prev) < n.minGap {
			n.mu.Unlock()
			return
		}
		n.lastSent[item.ChatID] = now
	}
	n.mu.Unlock()

	select {
	case n.ch <- item:
	default:
		// Channel full during burst; drop oldest and keep freshest.
		select {
		case <-n.ch:
		default:
		}
		select {
		case n.ch <- item:
		default:
		}
	}
}

// Close stops the background notification worker.
func (n *Notifier) Close() {
	if n == nil {
		return
	}
	n.once.Do(func() {
		close(n.ch)
	})
}

func (n *Notifier) loop() {
	for item := range n.ch {
		if n.sendFn != nil {
			n.sendFn(item.Title, item.Body)
		}
	}
}

func truncateClean(s string, maxRunes int) string {
	s = strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(s, "\r", " "), "\n", " "))
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes-1]) + "…"
}

func deliverNative(title, body string) {
	switch runtime.GOOS {
	case "darwin":
		esc := func(s string) string {
			s = strings.ReplaceAll(s, `\`, `\\`)
			return strings.ReplaceAll(s, `"`, `\"`)
		}
		script := fmt.Sprintf(`display notification "%s" with title "%s" sound name "Glass"`, esc(body), esc(title))
		cmd := exec.Command("osascript", "-e", script)
		configureHiddenCmd(cmd)
		_ = cmd.Run()

	case "windows":
		esc := func(s string) string {
			s = strings.ReplaceAll(s, "&", "&amp;")
			s = strings.ReplaceAll(s, "<", "&lt;")
			s = strings.ReplaceAll(s, ">", "&gt;")
			s = strings.ReplaceAll(s, "'", "''")
			return s
		}
		ps := fmt.Sprintf(
			`$t = '%s'; $b = '%s'; `+
				`try { `+
				`[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null; `+
				`[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null; `+
				`$xml = New-Object Windows.Data.Xml.Dom.XmlDocument; `+
				`$xml.LoadXml("<toast><visual><binding template='ToastText02'><text id='1'>$t</text><text id='2'>$b</text></binding></visual></toast>"); `+
				`$toast = [Windows.UI.Notifications.ToastNotification]::new($xml); `+
				`[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Parla Viva').Show($toast); `+
				`} catch {}`,
			esc(title), esc(body),
		)
		cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-WindowStyle", "Hidden", "-Command", ps)
		configureHiddenCmd(cmd)
		_ = cmd.Run()

	default:
		if p, err := exec.LookPath("notify-send"); err == nil {
			cmd := exec.Command(p, "-a", "Parla Viva", title, body)
			configureHiddenCmd(cmd)
			_ = cmd.Run()
		}
	}
}
