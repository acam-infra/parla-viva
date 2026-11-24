package appui

import (
	"image"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/vivaui/viva/gfx"
)

var nextBitmapID atomic.Uint64

// PendingAttachment represents one queued image or file waiting to be sent
// in a Telegram-style multi-attachment album.
type PendingAttachment struct {
	Path         string
	Name         string
	Size         int64
	IsImage      bool
	LoadingThumb bool
	Bitmap       gfx.Bitmap
}

// IsImagePath reports whether a filename has a supported raster image extension.
func IsImagePath(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".gif":
		return true
	default:
		return false
	}
}

// LoadBitmap decodes a local PNG/JPEG/GIF file into a Viva gfx.Bitmap (RGBA, scaled down if huge).
func LoadBitmap(path string, maxDim int) (gfx.Bitmap, bool) {
	f, err := os.Open(path)
	if err != nil {
		return gfx.Bitmap{}, false
	}
	defer f.Close()

	img, _, err := image.Decode(f)
	if err != nil {
		return gfx.Bitmap{}, false
	}
	return imageToBitmap(img, maxDim), true
}

// DemoGradientBitmap creates a crisp in-memory sample image bitmap for offline preview & tests.
func DemoGradientBitmap(w, h int, seed byte) gfx.Bitmap {
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			off := y*rgba.Stride + x*4
			rgba.Pix[off+0] = byte((x*255/max(w, 1) + int(seed)*37) % 256)
			rgba.Pix[off+1] = byte((y*255/max(h, 1) + int(seed)*73) % 256)
			rgba.Pix[off+2] = byte((180 + int(seed)*29) % 256)
			rgba.Pix[off+3] = 255
		}
	}
	return gfx.Bitmap{
		Pix:     rgba.Pix,
		Width:   w,
		Height:  h,
		Stride:  rgba.Stride,
		ID:      nextBitmapID.Add(1),
		Version: 1,
	}
}

func imageToBitmap(src image.Image, maxDim int) gfx.Bitmap {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return gfx.Bitmap{}
	}
	if maxDim > 0 && (w > maxDim || h > maxDim) {
		// Nearest-neighbor downscale to keep thumbnail memory bounded without extra deps.
		var nw, nh int
		if w >= h {
			nw = maxDim
			nh = max(h*maxDim/w, 1)
		} else {
			nh = maxDim
			nw = max(w*maxDim/h, 1)
		}
		dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
		for y := 0; y < nh; y++ {
			sy := b.Min.Y + y*h/nh
			for x := 0; x < nw; x++ {
				sx := b.Min.X + x*w/nw
				dst.Set(x, y, src.At(sx, sy))
			}
		}
		return gfx.Bitmap{
			Pix:     dst.Pix,
			Width:   nw,
			Height:  nh,
			Stride:  dst.Stride,
			ID:      nextBitmapID.Add(1),
			Version: 1,
		}
	}

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	return gfx.Bitmap{
		Pix:     rgba.Pix,
		Width:   w,
		Height:  h,
		Stride:  rgba.Stride,
		ID:      nextBitmapID.Add(1),
		Version: 1,
	}
}
