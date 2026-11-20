package preview

import (
	"strings"
	"testing"
)

func TestParseCommonFormats(t *testing.T) {
	// JSON pretty-print
	j := ParseBytes("config.json", []byte(`{"app":"parla-viva","v":3}`), 0, false)
	if j.Kind != KindJSON || !strings.Contains(j.Text, "\n  \"app\": \"parla-viva\"") {
		t.Fatalf("JSON preview failed: %+v", j)
	}

	// CSV table parsing
	c := ParseBytes("metrics.csv", []byte("Platform,SizeMB,Status\nWindows,2.76,OK\nmacOS,2.42,OK\n"), 0, false)
	if c.Kind != KindCSV || len(c.Headers) != 3 || len(c.Rows) != 2 || c.Rows[0][0] != "Windows" {
		t.Fatalf("CSV preview failed: %+v", c)
	}

	// Markdown
	m := ParseBytes("NOTES.md", []byte("# Release v0.3.0\n\n- Unified UI\n"), 0, false)
	if m.Kind != KindMarkdown || !strings.Contains(m.Text, "Release v0.3.0") {
		t.Fatalf("Markdown preview failed: %+v", m)
	}

	// vCard
	v := ParseBytes("alice.vcf", []byte("BEGIN:VCARD\nFN:Alice Dev\nEMAIL:alice@nine.testrun.org\nORG:Parla Viva\nEND:VCARD"), 0, false)
	if v.Kind != KindVCard || v.VCard == nil || v.VCard.FullName != "Alice Dev" || v.VCard.Email != "alice@nine.testrun.org" {
		t.Fatalf("vCard preview failed: %+v", v)
	}

	// Code with line numbers
	code := ParseBytes("main.go", []byte("package main\nfunc main() {}\n"), 0, false)
	if code.Kind != KindCode || !strings.Contains(code.Text, "   1 | package main") {
		t.Fatalf("Code preview failed: %+v", code)
	}
}
