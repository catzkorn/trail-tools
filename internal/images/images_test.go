package images

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/color"
	"math"
	"runtime"
	"strings"
	"testing"

	"golang.org/x/image/draw"
)

func TestSquare(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		name       string
		w, h, size int
		want       int
	}{
		{"landscape", 400, 200, 100, 100},
		{"portrait", 200, 400, 100, 100},
		{"smaller than size", 60, 80, 100, 60},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			img := image.NewRGBA(image.Rect(0, 0, tt.w, tt.h))
			// Mark the middle, which the square keeps.
			img.Set(tt.w/2, tt.h/2, color.RGBA{R: 255, A: 255})
			got := Square(img, tt.size)
			if b := got.Bounds(); b.Dx() != tt.want || b.Dy() != tt.want {
				t.Errorf("Square(%dx%d, %d) is %dx%d, want %dx%d", tt.w, tt.h, tt.size, b.Dx(), b.Dy(), tt.want, tt.want)
			}
		})
	}
}

func TestDecodeRejectsNonImages(t *testing.T) {
	t.Parallel()
	if _, err := Decode([]byte("not an image")); err != ErrNotImage {
		t.Errorf("Decode returned %v, want ErrNotImage", err)
	}
}

// TestScaleDownLargeImage scales down a photo of MaxPixels pixels, which
// draw.CatmullRom.Scale alone needs over 300 MB of buffer for.
func TestScaleDownLargeImage(t *testing.T) {
	// Not parallel, so that the measured allocations are only this test's.
	const w, h = 8000, 5000
	img := image.NewYCbCr(image.Rect(0, 0, w, h), image.YCbCrSubsampleRatio420)
	// The left half is red and the right half blue.
	for y := range h {
		for x := range w {
			c := color.RGBA{R: 255, A: 255}
			if x >= w/2 {
				c = color.RGBA{B: 255, A: 255}
			}
			yy, cb, cr := color.RGBToYCbCr(c.R, c.G, c.B)
			img.Y[img.YOffset(x, y)] = yy
			img.Cb[img.COffset(x, y)] = cb
			img.Cr[img.COffset(x, y)] = cr
		}
	}
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	got := ScaleDown(img, 2048)
	runtime.ReadMemStats(&after)
	if b := got.Bounds(); b.Dx() != 2048 || b.Dy() != 1280 {
		t.Fatalf("ScaleDown is %dx%d, want 2048x1280", b.Dx(), b.Dy())
	}
	if alloc := after.TotalAlloc - before.TotalAlloc; alloc > 150<<20 {
		t.Errorf("ScaleDown allocated %d MB, want at most 150 MB", alloc>>20)
	}
	// Every band of rows and columns is scaled.
	for _, p := range []image.Point{{0, 0}, {500, 640}, {1000, 1279}, {1100, 0}, {1600, 640}, {2047, 1279}} {
		r, g, b, _ := got.At(p.X, p.Y).RGBA()
		wantR, wantB := r > 0xe000, b < 0x2000
		if p.X > 1024 {
			wantR, wantB = r < 0x2000, b > 0xe000
		}
		if !wantR || !wantB || g > 0x2000 {
			t.Errorf("pixel %v is %v, want %s", p, got.At(p.X, p.Y), map[bool]string{true: "red", false: "blue"}[p.X < 1024])
		}
	}
}

// TestCatmullRomBands checks that scaling in bands matches scaling at once.
func TestCatmullRomBands(t *testing.T) {
	t.Parallel()
	// Large enough for several bands of rows and of columns, with smooth
	// gradients and a sharp edge.
	src := image.NewNRGBA(image.Rect(0, 0, 1500, 1200))
	for y := range 1200 {
		for x := range 1500 {
			v := uint8(128 + 100*math.Sin(float64(x)/40)*math.Cos(float64(y)/60))
			if x > 700 && x < 800 {
				v = 0
			}
			src.Set(x, y, color.NRGBA{R: v, G: uint8(x * 255 / 1500), B: uint8(y * 255 / 1200), A: uint8(155 + y%100)})
		}
	}
	dr := image.Rect(0, 0, 1000, 800)
	want := image.NewRGBA(dr)
	draw.Draw(want, dr, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.CatmullRom.Scale(want, dr, src, src.Bounds(), draw.Over, nil)
	got := image.NewRGBA(dr)
	draw.Draw(got, dr, image.NewUniform(color.White), image.Point{}, draw.Src)
	catmullRom(got, src, src.Bounds())
	maxDiff := 0
	for i := range got.Pix {
		maxDiff = max(maxDiff, int(got.Pix[i])-int(want.Pix[i]), int(want.Pix[i])-int(got.Pix[i]))
	}
	if maxDiff > 3 {
		t.Errorf("scaling in bands differs from scaling at once by up to %d", maxDiff)
	}
}

// pngHeader returns the start of a PNG of w by h pixels with the bit depth
// and color type, enough for image.DecodeConfig.
func pngHeader(w, h int, depth, colorType byte) []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], uint32(w))
	binary.BigEndian.PutUint32(ihdr[4:], uint32(h))
	ihdr[8], ihdr[9] = depth, colorType
	var b bytes.Buffer
	b.WriteString("\x89PNG\r\n\x1a\n")
	_ = binary.Write(&b, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	b.Write(chunk)
	_ = binary.Write(&b, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return b.Bytes()
}

func TestDecodeLimitsMemory(t *testing.T) {
	t.Parallel()
	// 30 megapixels of 16 bit RGBA would decode to 240 MB.
	_, err := Decode(pngHeader(6000, 5000, 16, 6))
	if err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("Decode of a 30 megapixel 16 bit PNG returned %v, want it to be too large", err)
	}
	// 30 megapixels of 8 bit RGBA is allowed, but this PNG has no data.
	if _, err := Decode(pngHeader(6000, 5000, 8, 6)); err != ErrNotImage {
		t.Errorf("Decode of a 30 megapixel 8 bit PNG without data returned %v, want ErrNotImage", err)
	}
}
