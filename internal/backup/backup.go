package backup

import (
	"archive/zip"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	magicHeader = "PARLAVIVA1"
	saltSize    = 16
	nonceSize   = 12
	kdfRounds   = 100_000
)

// ManifestEntry records a single file's relative path, size, and SHA-256 digest.
type ManifestEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
}

// Info summarizes a verified or created encrypted backup archive.
type Info struct {
	FormatVersion int             `json:"formatVersion"`
	CreatedAt     string          `json:"createdAt"`
	FileCount     int             `json:"fileCount"`
	TotalBytes    int64           `json:"totalBytes"`
	Entries       []ManifestEntry `json:"entries"`
}

// Create packages srcDir into an authenticated AES-256-GCM encrypted backup file at dstPath.
// Existing destination files are never overwritten (mirroring Parla SafeStore policy).
func Create(srcDir, dstPath, password string) (Info, error) {
	if strings.TrimSpace(password) == "" {
		return Info{}, errors.New("backup: password must not be empty")
	}
	if _, err := os.Stat(dstPath); err == nil {
		return Info{}, fmt.Errorf("backup: destination already exists: %s", dstPath)
	}

	var zipBuf bytes.Buffer
	zw := zip.NewWriter(&zipBuf)
	info := Info{
		FormatVersion: 1,
		CreatedAt:     time.Now().UTC().Format(time.RFC3339),
	}

	err := filepath.WalkDir(srcDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		rel, err := filepath.Rel(srcDir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		raw, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(raw)
		w, err := zw.Create("data/" + rel)
		if err != nil {
			return err
		}
		if _, err := w.Write(raw); err != nil {
			return err
		}
		info.Entries = append(info.Entries, ManifestEntry{
			Path:   rel,
			Size:   int64(len(raw)),
			SHA256: hex.EncodeToString(sum[:]),
		})
		info.FileCount++
		info.TotalBytes += int64(len(raw))
		return nil
	})
	if err != nil {
		_ = zw.Close()
		return Info{}, err
	}

	manifestBytes, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		_ = zw.Close()
		return Info{}, err
	}
	mw, err := zw.Create("manifest.json")
	if err != nil {
		_ = zw.Close()
		return Info{}, err
	}
	if _, err := mw.Write(manifestBytes); err != nil {
		_ = zw.Close()
		return Info{}, err
	}
	if err := zw.Close(); err != nil {
		return Info{}, err
	}

	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return Info{}, err
	}
	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return Info{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return Info{}, err
	}
	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return Info{}, err
	}

	ciphertext := gcm.Seal(nil, nonce, zipBuf.Bytes(), []byte(magicHeader))
	var out bytes.Buffer
	out.WriteString(magicHeader)
	out.Write(salt)
	out.Write(nonce)
	out.Write(ciphertext)

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o700); err != nil {
		return Info{}, err
	}
	tmp := dstPath + ".tmp"
	if err := os.WriteFile(tmp, out.Bytes(), 0o600); err != nil {
		return Info{}, err
	}
	if err := os.Rename(tmp, dstPath); err != nil {
		_ = os.Remove(tmp)
		return Info{}, err
	}
	return info, nil
}

// Verify decrypts and validates the manifest and every member's SHA-256 digest without writing files.
func Verify(archivePath, password string) (Info, error) {
	zr, info, err := openAndVerify(archivePath, password)
	if err != nil {
		return Info{}, err
	}
	_ = zr
	return info, nil
}

// Restore decrypts, verifies, and extracts an archive into a fresh target directory.
func Restore(archivePath, targetDir, password string) (Info, error) {
	zr, info, err := openAndVerify(archivePath, password)
	if err != nil {
		return Info{}, err
	}
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return Info{}, err
	}
	filesByPath := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		filesByPath[f.Name] = f
	}
	for _, entry := range info.Entries {
		if strings.Contains(entry.Path, "..") || strings.HasPrefix(entry.Path, "/") {
			return Info{}, fmt.Errorf("backup: unsafe entry path %q", entry.Path)
		}
		zf, ok := filesByPath["data/"+entry.Path]
		if !ok {
			return Info{}, fmt.Errorf("backup: missing member %q", entry.Path)
		}
		rc, err := zf.Open()
		if err != nil {
			return Info{}, err
		}
		raw, err := io.ReadAll(rc)
		_ = rc.Close()
		if err != nil {
			return Info{}, err
		}
		dest := filepath.Join(targetDir, filepath.FromSlash(entry.Path))
		if err := os.MkdirAll(filepath.Dir(dest), 0o700); err != nil {
			return Info{}, err
		}
		if err := os.WriteFile(dest, raw, 0o600); err != nil {
			return Info{}, err
		}
	}
	return info, nil
}

func openAndVerify(archivePath, password string) (*zip.Reader, Info, error) {
	raw, err := os.ReadFile(archivePath)
	if err != nil {
		return nil, Info{}, err
	}
	minLen := len(magicHeader) + saltSize + nonceSize + 16
	if len(raw) < minLen || string(raw[:len(magicHeader)]) != magicHeader {
		return nil, Info{}, errors.New("backup: invalid archive header")
	}
	off := len(magicHeader)
	salt := raw[off : off+saltSize]
	off += saltSize
	nonce := raw[off : off+nonceSize]
	off += nonceSize
	ciphertext := raw[off:]

	key := deriveKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, Info{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, Info{}, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, []byte(magicHeader))
	if err != nil {
		return nil, Info{}, errors.New("backup: authentication failed (wrong password or corrupted archive)")
	}

	zr, err := zip.NewReader(bytes.NewReader(plain), int64(len(plain)))
	if err != nil {
		return nil, Info{}, err
	}
	filesByPath := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		filesByPath[f.Name] = f
	}
	mf, ok := filesByPath["manifest.json"]
	if !ok {
		return nil, Info{}, errors.New("backup: missing manifest.json")
	}
	rc, err := mf.Open()
	if err != nil {
		return nil, Info{}, err
	}
	mBytes, err := io.ReadAll(rc)
	_ = rc.Close()
	if err != nil {
		return nil, Info{}, err
	}
	var info Info
	if err := json.Unmarshal(mBytes, &info); err != nil {
		return nil, Info{}, err
	}
	for _, entry := range info.Entries {
		zf, ok := filesByPath["data/"+entry.Path]
		if !ok {
			return nil, Info{}, fmt.Errorf("backup: missing member %q", entry.Path)
		}
		erc, err := zf.Open()
		if err != nil {
			return nil, Info{}, err
		}
		data, err := io.ReadAll(erc)
		_ = erc.Close()
		if err != nil {
			return nil, Info{}, err
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != entry.SHA256 {
			return nil, Info{}, fmt.Errorf("backup: hash mismatch for %q", entry.Path)
		}
	}
	return zr, info, nil
}

// deriveKey stretches password+salt into a 32-byte AES-256 key via iterated SHA-256.
func deriveKey(password string, salt []byte) []byte {
	h := sha256.New()
	h.Write(salt)
	h.Write([]byte(password))
	dk := h.Sum(nil)
	for i := 1; i < kdfRounds; i++ {
		h.Reset()
		h.Write(dk)
		h.Write(salt)
		dk = h.Sum(dk[:0])
	}
	return dk
}
