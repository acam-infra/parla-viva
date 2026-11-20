package webxdc

import (
	"archive/zip"
	"bytes"
)

// DemoChecklistArchive builds an in-memory interactive .xdc ZIP bundle for offline preview.
func DemoChecklistArchive() (*Archive, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	files := map[string]string{
		"manifest.toml": "name = \"Release Checklist\"\n",
		"index.html": `<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>Release Checklist</title>
<script src="webxdc.js"></script>
<style>body{font-family:system-ui,sans-serif;margin:16px;color:#111}li{margin:8px 0}</style>
</head><body>
<h3>Parla Viva v0.1.0 Checklist</h3>
<ul>
  <li><label><input type="checkbox" checked> Stdio JSON-RPC transport</label></li>
  <li><label><input type="checkbox" checked> 61-message bounded history window</label></li>
  <li><label><input type="checkbox" checked> Offline Webxdc CSP sandbox</label></li>
  <li><label><input type="checkbox" checked> SafeStore AES-256-GCM backups</label></li>
</ul>
</body></html>`,
	}
	for name, content := range files {
		w, err := zw.Create(name)
		if err != nil {
			return nil, err
		}
		if _, err := w.Write([]byte(content)); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return OpenBytes(buf.Bytes())
}
