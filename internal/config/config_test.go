package config

import (
	"path/filepath"
	"testing"
)

func TestConfigLoadSaveAndSetKey(t *testing.T) {
	dir := t.TempDir()
	cfg, path, err := Load(dir)
	if err != nil {
		t.Fatalf("Load default: %v", err)
	}
	if path != filepath.Join(dir, DefaultFilename) {
		t.Fatalf("unexpected path: %s", path)
	}
	if cfg.LogLevel != "info" || !cfg.Notifications {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}

	if err := cfg.SetKey("theme", "dark"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetKey("logLevel", "debug"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetKey("notifications", "false"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetKey("theme", "neon"); err == nil {
		t.Fatal("expected error for invalid theme")
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}
	reloaded, _, err := Load(dir)
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if reloaded.Theme != "dark" || reloaded.LogLevel != "debug" || reloaded.Notifications {
		t.Fatalf("reloaded mismatch: %+v", reloaded)
	}
}
