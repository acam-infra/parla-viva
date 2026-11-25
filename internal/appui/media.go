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

// BitmapFromRGBA converts straight RGBA pixel data into Viva's premultiplied BGRA gfx.Bitmap.
func BitmapFromRGBA(rgba []byte, w, h int) gfx.Bitmap {
	if w <= 0 || h <= 0 || len(rgba) < w*h*4 {
		return gfx.Bitmap{}
	}
	pix := make([]byte, w*h*4)
	for i := 0; i < w*h*4; i += 4 {
		r, g, b, a := rgba[i], rgba[i+1], rgba[i+2], rgba[i+3]
		if a < 255 {
			r = byte((uint16(r)*uint16(a) + 127) / 255)
			g = byte((uint16(g)*uint16(a) + 127) / 255)
			b = byte((uint16(b)*uint16(a) + 127) / 255)
		}
		pix[i+0] = b
		pix[i+1] = g
		pix[i+2] = r
		pix[i+3] = a
	}
	return gfx.Bitmap{
		Pix:     pix,
		Width:   w,
		Height:  h,
		Stride:  w * 4,
		ID:      nextBitmapID.Add(1),
		Version: 1,
	}
}

// DemoGradientBitmap creates a crisp in-memory sample sunset/mountain image bitmap for offline preview & tests.
func DemoGradientBitmap(w, h int, seed byte) gfx.Bitmap {
	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		t := float64(y) / float64(max(h-1, 1))
		for x := 0; x < w; x++ {
			s := float64(x) / float64(max(w-1, 1))
			r := byte(min(255, int(44+120*(1-t)+45*s+float64(seed)*7)))
			g := byte(min(255, int(52+78*(1-t)+30*s)))
			b := byte(min(255, int(92+68*t+30*(1-s))))
			// Subtle warm sun disc in upper-right quadrant
			dx := s - 0.76
			dy := t - 0.30
			if dx*dx*1.8+dy*dy < 0.022 {
				r, g, b = 250, 214, 134
			}
			// Mountain ridge silhouette near bottom
			ridge := 0.64 + 0.08*float64((x*3+int(seed)*17)%19)/19.0
			if s < 0.45 {
				ridge -= (0.45 - s) * 0.25
			} else {
				ridge -= (s - 0.45) * 0.14
			}
			if t > ridge {
				r, g, b = 36, 44, 66
			}
			off := y*rgba.Stride + x*4
			rgba.Pix[off+0] = r
			rgba.Pix[off+1] = g
			rgba.Pix[off+2] = b
			rgba.Pix[off+3] = 255
		}
	}
	return BitmapFromRGBA(rgba.Pix, w, h)
}

func imageToBitmap(src image.Image, maxDim int) gfx.Bitmap {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return gfx.Bitmap{}
	}
	if maxDim > 0 && (w > maxDim || h > maxDim) {
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
		return BitmapFromRGBA(dst.Pix, nw, nh)
	}

	rgba := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(rgba, rgba.Bounds(), src, b.Min, draw.Src)
	return BitmapFromRGBA(rgba.Pix, w, h)
}
