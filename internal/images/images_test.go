package images

import (
	"image"
	"image/color"
	"testing"
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
