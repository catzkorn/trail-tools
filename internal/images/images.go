// Package images decodes uploaded images and encodes the stored JPEGs.
package images

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"

	// Register the decoders of the accepted formats.
	_ "image/png"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

// MaxPixels limits the size of decoded images, to bound memory.
const MaxPixels = 40_000_000

// ErrNotImage is returned for uploads that aren't a supported image.
var ErrNotImage = errors.New("the file is not a JPEG, PNG or WebP image")

// Decode decodes a JPEG, PNG or WebP image of at most MaxPixels pixels.
func Decode(data []byte) (image.Image, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, ErrNotImage
	}
	if cfg.Width*cfg.Height > MaxPixels {
		return nil, fmt.Errorf("the image is too large: %dx%d pixels, the maximum is %d megapixels", cfg.Width, cfg.Height, MaxPixels/1_000_000)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, ErrNotImage
	}
	return img, nil
}

// ScaleDown returns the image scaled to fit within size by size pixels, on
// a white background so that transparent areas aren't black in a JPEG.
func ScaleDown(img image.Image, size int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > size || h > size {
		if w >= h {
			w, h = size, max(1, h*size/w)
		} else {
			w, h = max(1, w*size/h), size
		}
	}
	return scale(img, b, w, h)
}

// Square returns the middle square of the image, scaled down to at most
// size by size pixels, on a white background.
func Square(img image.Image, size int) image.Image {
	b := img.Bounds()
	side := min(b.Dx(), b.Dy())
	x, y := b.Min.X+(b.Dx()-side)/2, b.Min.Y+(b.Dy()-side)/2
	crop := image.Rect(x, y, x+side, y+side)
	out := min(side, size)
	return scale(img, crop, out, out)
}

// scale draws the src rectangle of the image scaled to w by h pixels, on a
// white background.
func scale(img image.Image, src image.Rectangle, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	if w == src.Dx() && h == src.Dy() {
		draw.Draw(dst, dst.Bounds(), img, src.Min, draw.Over)
	} else {
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, src, draw.Over, nil)
	}
	return dst
}

// EncodeJPEG encodes an image as a JPEG. Encoding drops the metadata of the
// original upload, such as its GPS position.
func EncodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}
	return buf.Bytes(), nil
}
