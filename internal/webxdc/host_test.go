package webxdc

import (
	"archive/zip"
	"bytes"
	"strings"
	"testing"

	"github.com/vivaui/viva/app"
)

func buildTestXDC(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestArchiveRespondAndSecurityPolicy(t *testing.T) {
	raw := buildTestXDC(t, map[string]string{
		"index.html": `<!doctype html><script src="webxdc.js"></script><h1>Checklist</h1>`,
		"app.css":    `body { color: #111; }`,
	})
	arc, err := OpenBytes(raw)
	if err != nil {
		t.Fatalf("OpenBytes: %v", err)
	}

	resp, ok := arc.Respond(app.WebRequest{Path: "/index.html"})
	if !ok || resp.Status != 200 || !strings.Contains(string(resp.Data), "Checklist") {
		t.Fatalf("index.html response = %+v, ok=%v", resp, ok)
	}
	if !strings.Contains(resp.Headers["Content-Security-Policy"], "connect-src 'none'") {
		t.Fatalf("missing strict offline CSP header: %+v", resp.Headers)
	}

	jsResp, ok := arc.Respond(app.WebRequest{Path: "/webxdc.js"})
	if !ok || !strings.Contains(string(jsResp.Data), "window.webxdc") {
		t.Fatalf("expected built-in webxdc.js bridge, got %+v", jsResp)
	}

	if _, ok := arc.Respond(app.WebRequest{Path: "/../../etc/passwd"}); ok {
		t.Fatal("expected path traversal to be rejected")
	}

	cfg := arc.WebViewConfig(nil)
	if !cfg.Ephemeral || !cfg.BridgeMainFrameOnly {
		t.Fatal("expected Ephemeral and BridgeMainFrameOnly to be enabled")
	}
	if cfg.Navigate(app.WebNavigation{URL: "https://evil.example/"}) {
		t.Fatal("expected external navigation to be blocked")
	}
	if !cfg.Navigate(app.WebNavigation{URL: "webxdc://app/index.html"}) {
		t.Fatal("expected internal webxdc navigation to be allowed")
	}
}
