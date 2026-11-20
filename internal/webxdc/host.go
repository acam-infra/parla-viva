package webxdc

import (
	"archive/zip"
	"bytes"
	"errors"
	"io"
	"mime"
	"path"
	"strings"

	"github.com/vivaui/viva/app"
)

const (
	// SchemeName is the custom scheme used to serve offline Webxdc mini-apps.
	SchemeName = "webxdc"
	// strictCSP blocks all external network connections for offline Webxdc apps.
	strictCSP = "default-src 'self' 'unsafe-inline' 'unsafe-eval' data: blob:; connect-src 'none';"
)

// webxdcBridgeJS implements the core window.webxdc surface over Viva's window.viva bridge.
const webxdcBridgeJS = `(function(){
  if (window.webxdc) return;
  let listener = null;
  let serial = 0;
  window.webxdc = {
    selfAddr: "self@parla.local",
    selfName: "Parla User",
    sendUpdate: function(update, descr) {
      if (window.viva && window.viva.postMessage) {
        window.viva.postMessage(JSON.stringify({type: "sendUpdate", update: update, descr: descr || ""}));
      }
    },
    setUpdateListener: function(cb, startSerial) {
      listener = cb;
      serial = startSerial || 0;
      if (window.viva && window.viva.postMessage) {
        window.viva.postMessage(JSON.stringify({type: "setUpdateListener", serial: serial}));
      }
      return Promise.resolve();
    },
    getAllUpdates: function() { return Promise.resolve([]); }
  };
  if (window.viva) {
    window.viva.onmessage = function(raw) {
      try {
        const msg = JSON.parse(raw);
        if (msg.type === "update" && typeof listener === "function") {
          listener(msg.update);
        }
      } catch (_) {}
    };
  }
})();
`

// Archive holds an in-memory .xdc ZIP bundle ready to serve via app.WebScheme.
type Archive struct {
	files map[string][]byte
}

// OpenBytes parses a .xdc ZIP archive from memory.
func OpenBytes(data []byte) (*Archive, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, err
	}
	files := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		clean, err := sanitizePath(f.Name)
		if err != nil {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, err
		}
		b, err := io.ReadAll(io.LimitReader(rc, 32*1024*1024))
		_ = rc.Close()
		if err != nil {
			return nil, err
		}
		files[clean] = b
	}
	return &Archive{files: files}, nil
}

// Respond serves a single WebRequest from the .xdc archive with strict offline CSP.
func (a *Archive) Respond(req app.WebRequest) (app.WebResponse, bool) {
	clean, err := sanitizePath(req.Path)
	if err != nil {
		return app.WebResponse{Status: 403, MIME: "text/plain", Data: []byte("forbidden path")}, false
	}
	if clean == "" {
		clean = "index.html"
	}
	if clean == "webxdc.js" {
		return app.WebResponse{
			Status: 200,
			MIME:   "application/javascript; charset=utf-8",
			Headers: map[string]string{
				"Content-Security-Policy":     strictCSP,
				"Access-Control-Allow-Origin": "",
			},
			Data: []byte(webxdcBridgeJS),
		}, true
	}
	data, ok := a.files[clean]
	if !ok {
		return app.WebResponse{Status: 404, MIME: "text/plain", Data: []byte("not found")}, false
	}
	contentType := mime.TypeByExtension(path.Ext(clean))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return app.WebResponse{
		Status: 200,
		MIME:   contentType,
		Headers: map[string]string{
			"Content-Security-Policy":     strictCSP,
			"Access-Control-Allow-Origin": "",
		},
		Data: data,
	}, true
}

// WebViewConfig builds a hardened Viva WebViewConfig for running this offline Webxdc app.
func (a *Archive) WebViewConfig(onMessage func(string)) app.WebViewConfig {
	return app.WebViewConfig{
		URL:                 SchemeName + "://app/index.html",
		Ephemeral:           true,
		BridgeMainFrameOnly: true,
		OnMessage:           onMessage,
		Scheme: &app.WebScheme{
			Name:    SchemeName,
			Respond: a.Respond,
		},
		Navigate: func(nav app.WebNavigation) bool {
			return strings.HasPrefix(nav.URL, SchemeName+"://app/")
		},
		NewWindow: func(string) bool { return false },
		Download:  func(string) bool { return false },
	}
}

func sanitizePath(raw string) (string, error) {
	p := strings.TrimPrefix(raw, "/")
	if strings.Contains(p, "\\") || strings.Contains(p, "\x00") {
		return "", errors.New("invalid path separator")
	}
	for _, seg := range strings.Split(p, "/") {
		if seg == ".." {
			return "", errors.New("path traversal")
		}
	}
	cleaned := path.Clean("/" + p)
	cleaned = strings.TrimPrefix(cleaned, "/")
	if cleaned == "." {
		return "", nil
	}
	return cleaned, nil
}
