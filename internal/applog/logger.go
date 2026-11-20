package applog

import (
	"bufio"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Logger wraps slog.Logger with size-based file rotation and safe cleanup.
type Logger struct {
	*slog.Logger
	writer *rotatingWriter
	path   string
}

// Open creates a structured logger writing to logPath with maxMB rotation.
// If logPath is empty, logs are discarded cleanly.
func Open(logPath, levelStr string, maxMB int) (*Logger, error) {
	level := parseLevel(levelStr)
	if logPath == "" {
		h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: level})
		return &Logger{Logger: slog.New(h)}, nil
	}
	if maxMB <= 0 {
		maxMB = 10
	}
	if err := os.MkdirAll(filepath.Dir(logPath), 0o700); err != nil {
		return nil, err
	}
	rw, err := newRotatingWriter(logPath, int64(maxMB)*1024*1024)
	if err != nil {
		return nil, err
	}
	h := slog.NewTextHandler(rw, &slog.HandlerOptions{Level: level})
	l := slog.New(h)
	slog.SetDefault(l)
	return &Logger{Logger: l, writer: rw, path: logPath}, nil
}

// Path returns the active log file path.
func (l *Logger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

// Close flushes and closes the underlying log file.
func (l *Logger) Close() error {
	if l == nil || l.writer == nil {
		return nil
	}
	return l.writer.Close()
}

// TailLines returns up to n most recent lines from logPath, optionally filtered
// by a case-insensitive substring (such as "ERROR" or "WARN").
func TailLines(logPath string, n int, filter string) ([]string, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	if n <= 0 {
		n = 50
	}
	want := strings.ToLower(strings.TrimSpace(filter))
	var lines []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if want != "" && !strings.Contains(strings.ToLower(line), want) {
			continue
		}
		lines = append(lines, line)
		if len(lines) > n*2 {
			lines = lines[len(lines)-n:]
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines, nil
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

type rotatingWriter struct {
	mu       sync.Mutex
	path     string
	maxBytes int64
	file     *os.File
	size     int64
}

func newRotatingWriter(path string, maxBytes int64) (*rotatingWriter, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	var sz int64
	if info, err := f.Stat(); err == nil {
		sz = info.Size()
	}
	return &rotatingWriter{path: path, maxBytes: maxBytes, file: f, size: sz}, nil
}

func (w *rotatingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return 0, os.ErrClosed
	}
	if w.maxBytes > 0 && w.size+int64(len(p)) > w.maxBytes {
		_ = w.rotateLocked()
	}
	n, err := w.file.Write(p)
	w.size += int64(n)
	return n, err
}

func (w *rotatingWriter) rotateLocked() error {
	_ = w.file.Close()
	backup := w.path + ".1"
	_ = os.Remove(backup)
	_ = os.Rename(w.path, backup)
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		w.file = nil
		return err
	}
	w.file = f
	w.size = 0
	return nil
}

func (w *rotatingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
