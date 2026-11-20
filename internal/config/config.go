package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/acam-infra/parla-viva/internal/dc"
)

const DefaultFilename = "parla-viva.json"

// Config holds user-editable Parla Viva runtime preferences and logging settings.
type Config struct {
	AccountsDir       string `json:"accountsDir,omitempty"`
	RPCServer         string `json:"rpcServer,omitempty"`
	Theme             string `json:"theme"`             // "light", "dark", "system"
	Layout            string `json:"layout"`            // "bubbles", "workspace", "compact"
	EnterToSend       bool   `json:"enterToSend"`       // Enter submits message
	Notifications     bool   `json:"notifications"`     // Native OS notifications
	NotificationSound bool   `json:"notificationSound"` // Notification sound on macOS/Windows
	NotifyThrottleMs  int    `json:"notifyThrottleMs"`  // Per-chat throttle in ms
	LogLevel          string `json:"logLevel"`          // "debug", "info", "warn", "error"
	LogFile           string `json:"logFile,omitempty"` // Defaults to <accountsDir>/logs/parla-viva.log
	LogMaxMB          int    `json:"logMaxMB"`          // Rotation threshold in MB
}

// Default returns sensible defaults for a given accounts directory.
func Default(accountsDir string) Config {
	if accountsDir == "" {
		accountsDir = dc.DefaultAccountsDir()
	}
	return Config{
		AccountsDir:       accountsDir,
		Theme:             "light",
		Layout:            "bubbles",
		EnterToSend:       true,
		Notifications:     true,
		NotificationSound: true,
		NotifyThrottleMs:  1500,
		LogLevel:          "info",
		LogFile:           filepath.Join(accountsDir, "logs", "parla-viva.log"),
		LogMaxMB:          10,
	}
}

// PathFor resolves the active config file path, honoring PARLA_VIVA_CONFIG.
func PathFor(accountsDir string) string {
	if env := strings.TrimSpace(os.Getenv("PARLA_VIVA_CONFIG")); env != "" {
		return env
	}
	if accountsDir == "" {
		accountsDir = dc.DefaultAccountsDir()
	}
	return filepath.Join(accountsDir, DefaultFilename)
}

// Load reads Config from path (or defaults if the file does not exist yet),
// applying environment overrides (PARLA_VIVA_LOG_LEVEL, PARLA_VIVA_NOTIFICATIONS).
func Load(accountsDir string) (Config, string, error) {
	cfg := Default(accountsDir)
	p := PathFor(accountsDir)
	raw, err := os.ReadFile(p)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return cfg, p, err
		}
	} else {
		if uerr := json.Unmarshal(raw, &cfg); uerr != nil {
			return cfg, p, fmt.Errorf("config: parse %s: %w", p, uerr)
		}
	}
	cfg.normalize(accountsDir)
	applyEnv(&cfg)
	return cfg, p, nil
}

// Save atomically writes Config to path with 0600 permissions.
func Save(path string, cfg Config) error {
	cfg.normalize(cfg.AccountsDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// SetKey updates a single config field by name (case-insensitive) and validates it.
func (c *Config) SetKey(key, val string) error {
	k := strings.ToLower(strings.TrimSpace(key))
	v := strings.TrimSpace(val)
	switch k {
	case "accountsdir", "accounts_dir", "accounts":
		c.AccountsDir = v
	case "rpcserver", "rpc_server", "rpc":
		c.RPCServer = v
	case "theme":
		switch strings.ToLower(v) {
		case "light", "dark", "system":
			c.Theme = strings.ToLower(v)
		default:
			return fmt.Errorf("invalid theme %q (use light, dark, or system)", v)
		}
	case "layout":
		switch strings.ToLower(v) {
		case "bubbles", "workspace", "compact":
			c.Layout = strings.ToLower(v)
		default:
			return fmt.Errorf("invalid layout %q (use bubbles, workspace, or compact)", v)
		}
	case "entertosend", "enter_to_send":
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %q", key, v)
		}
		c.EnterToSend = b
	case "notifications", "notify":
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %q", key, v)
		}
		c.Notifications = b
	case "notificationsound", "notification_sound", "sound":
		b, err := strconv.ParseBool(v)
		if err != nil {
			return fmt.Errorf("invalid boolean for %s: %q", key, v)
		}
		c.NotificationSound = b
	case "notifythrottlems", "notify_throttle_ms":
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			return fmt.Errorf("invalid non-negative integer for %s: %q", key, v)
		}
		c.NotifyThrottleMs = n
	case "loglevel", "log_level":
		switch strings.ToLower(v) {
		case "debug", "info", "warn", "error":
			c.LogLevel = strings.ToLower(v)
		default:
			return fmt.Errorf("invalid logLevel %q (use debug, info, warn, or error)", v)
		}
	case "logfile", "log_file":
		c.LogFile = v
	case "logmaxmb", "log_max_mb":
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid positive integer for %s: %q", key, v)
		}
		c.LogMaxMB = n
	default:
		return fmt.Errorf("unknown config key %q", key)
	}
	return nil
}

func (c *Config) normalize(accountsDir string) {
	if c.AccountsDir == "" {
		if accountsDir != "" {
			c.AccountsDir = accountsDir
		} else {
			c.AccountsDir = dc.DefaultAccountsDir()
		}
	}
	if c.Theme == "" {
		c.Theme = "light"
	}
	if c.Layout == "" {
		c.Layout = "bubbles"
	}
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if c.LogFile == "" {
		c.LogFile = filepath.Join(c.AccountsDir, "logs", "parla-viva.log")
	}
	if c.LogMaxMB <= 0 {
		c.LogMaxMB = 10
	}
	if c.NotifyThrottleMs <= 0 {
		c.NotifyThrottleMs = 1500
	}
}

func applyEnv(c *Config) {
	if v := strings.TrimSpace(os.Getenv("PARLA_VIVA_LOG_LEVEL")); v != "" {
		_ = c.SetKey("logLevel", v)
	}
	if v := strings.TrimSpace(os.Getenv("PARLA_VIVA_NOTIFICATIONS")); v != "" {
		_ = c.SetKey("notifications", v)
	}
}
