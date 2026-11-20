package applog

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestOpenWriteAndTailLines(t *testing.T) {
	p := filepath.Join(t.TempDir(), "logs", "parla-viva.log")
	l, err := Open(p, "debug", 1)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	l.Info("startup", "version", "0.7.0", "cacheHit", true)
	l.Warn("slow_rpc", "ms", 420)
	l.Error("sync_failed", "err", "timeout")
	_ = l.Close()

	all, err := TailLines(p, 10, "")
	if err != nil || len(all) != 3 {
		t.Fatalf("TailLines all = %d, err=%v", len(all), err)
	}
	errs, err := TailLines(p, 10, "ERROR")
	if err != nil || len(errs) != 1 || !strings.Contains(errs[0], "sync_failed") {
		t.Fatalf("TailLines ERROR = %v, err=%v", errs, err)
	}
}
