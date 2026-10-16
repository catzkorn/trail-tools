package trip

import (
	"github.com/johanbrandhorst/trail-tools/internal/images"
)

const (
	// maxPhotoBytes limits the size of uploaded photos. Browsers downscale
	// photos before uploading them, so they are usually much smaller.
	maxPhotoBytes = 20 << 20
	// largeSize and thumbSize are the maximum width and height of the stored
	// images.
	largeSize = 2048
	thumbSize = 480
)

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
	img, err := images.Decode(data)
	if err != nil {
		return nil, err
	}
	large := images.ScaleDown(img, largeSize)
	largeData, err := images.EncodeJPEG(large, 85)
	if err != nil {
		return nil, err
	}
	thumbData, err := images.EncodeJPEG(images.ScaleDown(large, thumbSize), 80)
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
