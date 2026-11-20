package preview

import (
	"archive/zip"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Kind identifies how a file should be rendered inside the universal in-app previewer.
type Kind string

const (
	KindImage    Kind = "Image"
	KindMarkdown Kind = "Markdown"
	KindJSON     Kind = "JSON"
	KindCSV      Kind = "Table"
	KindCode     Kind = "Code / Text"
	KindArchive  Kind = "Archive"
	KindVCard    Kind = "Contact Card"
	KindBinary   Kind = "File"
)

// ArchiveItem describes one file inside a ZIP or .xdc archive.
type ArchiveItem struct {
	Name string
	Size uint64
}

// VCardInfo holds parsed fields from a .vcf contact card.
type VCardInfo struct {
	FullName string
	Email    string
	Phone    string
	Org      string
	Note     string
}

// Document is the decoded in-app preview representation for any common file format.
type Document struct {
	Path         string
	Name         string
	Ext          string
	SizeBytes    int64
	Kind         Kind
	Text         string        // Raw/formatted text for Markdown, JSON, Code/Text
	Headers      []string      // Table headers for CSV/TSV
	Rows         [][]string    // Table rows for CSV/TSV
	ArchiveItems []ArchiveItem // Entries for ZIP/XDC
	VCard        *VCardInfo    // Contact fields for .vcf
	Truncated    bool
}

const maxPreviewBytes = 512 * 1024 // 512 KiB safety cap for in-app text/table rendering

// DetectKind classifies a filename by its extension.
func DetectKind(name string) Kind {
	ext := strings.ToLower(filepath.Ext(name))
	switch ext {
	case ".png", ".jpg", ".jpeg", ".gif":
		return KindImage
	case ".md", ".markdown":
		return KindMarkdown
	case ".json":
		return KindJSON
	case ".csv", ".tsv":
		return KindCSV
	case ".zip", ".xdc":
		return KindArchive
	case ".vcf":
		return KindVCard
	case ".txt", ".log", ".go", ".py", ".js", ".ts", ".tsx", ".jsx", ".rs", ".c", ".h", ".cpp",
		".java", ".sh", ".bash", ".ps1", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".xml", ".sql",
		".css", ".html", ".htm", ".env", ".diff", ".patch":
		return KindCode
	default:
		return KindBinary
	}
}

// LoadFile reads and parses a local file into a preview Document.
func LoadFile(path string) (Document, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Document{}, err
	}
	name := filepath.Base(path)
	kind := DetectKind(name)
	doc := Document{
		Path:      path,
		Name:      name,
		Ext:       strings.ToLower(filepath.Ext(name)),
		SizeBytes: info.Size(),
		Kind:      kind,
	}
	if kind == KindImage || kind == KindBinary {
		return doc, nil
	}
	if kind == KindArchive {
		zr, err := zip.OpenReader(path)
		if err != nil {
			doc.Kind = KindBinary
			return doc, nil
		}
		defer zr.Close()
		for i, f := range zr.File {
			if i >= 200 {
				doc.Truncated = true
				break
			}
			doc.ArchiveItems = append(doc.ArchiveItems, ArchiveItem{
				Name: f.Name,
				Size: f.UncompressedSize64,
			})
		}
		return doc, nil
	}

	f, err := os.Open(path)
	if err != nil {
		return Document{}, err
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, maxPreviewBytes+1))
	if err != nil {
		return Document{}, err
	}
	if len(raw) > maxPreviewBytes {
		raw = raw[:maxPreviewBytes]
		doc.Truncated = true
	}
	return ParseBytes(name, raw, doc.SizeBytes, doc.Truncated), nil
}

// ParseBytes parses raw file bytes into a preview Document (works for both local files and demo fixtures).
func ParseBytes(name string, raw []byte, totalSize int64, truncated bool) Document {
	if totalSize == 0 {
		totalSize = int64(len(raw))
	}
	kind := DetectKind(name)
	doc := Document{
		Name:      name,
		Ext:       strings.ToLower(filepath.Ext(name)),
		SizeBytes: totalSize,
		Kind:      kind,
		Truncated: truncated,
	}

	switch kind {
	case KindMarkdown:
		doc.Text = string(raw)
	case KindJSON:
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err == nil {
			doc.Text = buf.String()
		} else {
			doc.Text = string(raw)
		}
	case KindCSV:
		r := csv.NewReader(bytes.NewReader(raw))
		if doc.Ext == ".tsv" {
			r.Comma = '\t'
		}
		r.FieldsPerRecord = -1
		records, err := r.ReadAll()
		if err == nil && len(records) > 0 {
			doc.Headers = records[0]
			if len(records) > 1 {
				rows := records[1:]
				if len(rows) > 200 {
					rows = rows[:200]
					doc.Truncated = true
				}
				doc.Rows = rows
			}
		} else {
			doc.Kind = KindCode
			doc.Text = string(raw)
		}
	case KindVCard:
		doc.VCard = parseVCard(string(raw))
		doc.Text = string(raw)
	case KindArchive:
		if zr, err := zip.NewReader(bytes.NewReader(raw), int64(len(raw))); err == nil {
			for i, f := range zr.File {
				if i >= 200 {
					doc.Truncated = true
					break
				}
				doc.ArchiveItems = append(doc.ArchiveItems, ArchiveItem{
					Name: f.Name,
					Size: f.UncompressedSize64,
				})
			}
		}
	case KindCode:
		doc.Text = addLineNumbers(string(raw))
	}
	return doc
}

func parseVCard(raw string) *VCardInfo {
	vc := &VCardInfo{}
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r\n", "\n"), "\n") {
		line = strings.TrimSpace(line)
		upper := strings.ToUpper(line)
		idx := strings.IndexByte(line, ':')
		if idx < 0 {
			continue
		}
		val := strings.TrimSpace(line[idx+1:])
		switch {
		case strings.HasPrefix(upper, "FN"):
			vc.FullName = val
		case strings.HasPrefix(upper, "EMAIL"):
			if vc.Email == "" {
				vc.Email = val
			}
		case strings.HasPrefix(upper, "TEL"):
			if vc.Phone == "" {
				vc.Phone = val
			}
		case strings.HasPrefix(upper, "ORG"):
			vc.Org = val
		case strings.HasPrefix(upper, "NOTE"):
			vc.Note = val
		}
	}
	return vc
}

func addLineNumbers(s string) string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	if len(lines) > 400 {
		lines = lines[:400]
	}
	var b strings.Builder
	for i, l := range lines {
		if i > 0 {
			b.WriteByte('\n')
		}
		fmt.Fprintf(&b, "%4d | %s", i+1, l)
	}
	return b.String()
}
