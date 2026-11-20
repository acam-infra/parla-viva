package backup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateVerifyRestoreAndTamper(t *testing.T) {
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "acct1"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "acct1", "dc.db"), []byte("encrypted-sqlite-payload"), 0o600); err != nil {
		t.Fatal(err)
	}

	archive := filepath.Join(t.TempDir(), "parla-backup.pvz")
	info, err := Create(src, archive, "correct-horse-battery-staple")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if info.FileCount != 1 || info.TotalBytes != int64(len("encrypted-sqlite-payload")) {
		t.Fatalf("unexpected info: %+v", info)
	}

	// Refuse overwrite of existing backup.
	if _, err := Create(src, archive, "correct-horse-battery-staple"); err == nil {
		t.Fatal("expected overwrite refusal")
	}

	// Wrong password fails authentication.
	if _, err := Verify(archive, "wrong-password"); err == nil {
		t.Fatal("expected wrong password failure")
	}

	// Valid verify passes.
	verified, err := Verify(archive, "correct-horse-battery-staple")
	if err != nil || verified.FileCount != 1 {
		t.Fatalf("Verify: %+v, err=%v", verified, err)
	}

	// Restore recreates directory cleanly.
	restored := filepath.Join(t.TempDir(), "restored")
	if _, err := Restore(archive, restored, "correct-horse-battery-staple"); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(restored, "acct1", "dc.db"))
	if err != nil || string(got) != "encrypted-sqlite-payload" {
		t.Fatalf("restored content = %q, err=%v", got, err)
	}
}
