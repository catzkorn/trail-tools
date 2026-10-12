package trip

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

const (
	// maxPhotoBytes limits the size of uploaded photos. Browsers downscale
	// photos before uploading them, so they are usually much smaller.
	maxPhotoBytes = 20 << 20
	// maxPhotoPixels limits the size of decoded photos, to bound memory.
	maxPhotoPixels = 40_000_000
	// largeSize and thumbSize are the maximum width and height of the stored
	// images.
	largeSize = 2048
	thumbSize = 480
)

// errNotImage is returned for uploads that aren't a supported image.
var errNotImage = errors.New("the file is not a JPEG, PNG or WebP image")

// processedImage is an uploaded photo converted to the stored images.
type processedImage struct {
	large         []byte
	width, height int
	thumb         []byte
}

// processImage decodes an uploaded image and encodes it as a JPEG of at
// most largeSize pixels and a thumbnail of at most thumbSize pixels.
// Re-encoding drops the metadata of the upload, such as its GPS position.
func processImage(data []byte) (*processedImage, error) {
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, errNotImage
	}
	if cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, errNotImage
	}
	if cfg.Width*cfg.Height > maxPhotoPixels {
		return nil, fmt.Errorf("the image is too large: %dx%d pixels, the maximum is %d megapixels", cfg.Width, cfg.Height, maxPhotoPixels/1_000_000)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, errNotImage
	}
	large := scaleDown(img, largeSize)
	largeData, err := encodeJPEG(large, 85)
	if err != nil {
		return nil, err
	}
	thumbData, err := encodeJPEG(scaleDown(large, thumbSize), 80)
	if err != nil {
		return nil, err
	}
	return &processedImage{
		large:  largeData,
		width:  large.Bounds().Dx(),
		height: large.Bounds().Dy(),
		thumb:  thumbData,
	}, nil
}

// scaleDown returns the image scaled to fit within size by size pixels, on
// a white background so that transparent areas aren't black in a JPEG.
func scaleDown(img image.Image, size int) image.Image {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w > size || h > size {
		if w >= h {
			w, h = size, max(1, h*size/w)
		} else {
			w, h = max(1, w*size/h), size
		}
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(dst, dst.Bounds(), image.NewUniform(color.White), image.Point{}, draw.Src)
	if w == b.Dx() && h == b.Dy() {
		draw.Draw(dst, dst.Bounds(), img, b.Min, draw.Over)
	} else {
		draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Over, nil)
	}
	return dst
}

func encodeJPEG(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, fmt.Errorf("failed to encode JPEG: %w", err)
	}
	return buf.Bytes(), nil
}
