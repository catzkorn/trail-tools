package fakestrava

import (
	"bytes"
	"image"
	"image/jpeg"
	"math"
	"math/rand/v2"
)

// The size of the generated photos.
const (
	photoWidth  = 1600
	photoHeight = 1200
)

// landscape returns a JPEG of a made-up mountain landscape: a sky, layers of
// ridges fading into the distance, and a meadow in front.
func landscape(seed uint64) ([]byte, error) {
	r := rand.New(rand.NewPCG(seed, 7))
	img := image.NewRGBA(image.Rect(0, 0, photoWidth, photoHeight))

	// The sky, from deep blue at the top to pale near the horizon, or a
	// sunset now and then.
	top, horizon := [3]float64{70, 130, 200}, [3]float64{200, 225, 240}
	if r.Float64() < 0.25 {
		top, horizon = [3]float64{80, 90, 160}, [3]float64{250, 180, 120}
	}
	type ridge struct {
		base, amplitude float64
		phases          [3]float64
		freqs           [3]float64
		color           [3]float64
		snow            bool
	}
	var ridges []ridge
	layers := 3 + r.IntN(2)
	for i := range layers {
		depth := float64(i) / float64(layers-1)
		shade := 0.45 + 0.4*(1-depth)
		ridges = append(ridges, ridge{
			base:      0.35 + 0.15*depth + 0.05*r.Float64(),
			amplitude: 0.18 * (1 - 0.5*depth),
			phases:    [3]float64{r.Float64() * 6, r.Float64() * 6, r.Float64() * 6},
			freqs:     [3]float64{1 + r.Float64(), 3 + 2*r.Float64(), 9 + 4*r.Float64()},
			color:     [3]float64{60 + 80*shade, 80 + 70*shade, 90 + 80*shade},
			snow:      i == 0,
		})
	}
	// The meadow in front.
	ridges = append(ridges, ridge{
		base: 0.78, amplitude: 0.06,
		phases: [3]float64{r.Float64() * 6, r.Float64() * 6, 0},
		freqs:  [3]float64{0.7, 2, 0},
		color:  [3]float64{70 + 30*r.Float64(), 120 + 30*r.Float64(), 50},
	})

	// The height of each ridge at each column, as a fraction from the top.
	heights := make([][]float64, len(ridges))
	for i, rd := range ridges {
		heights[i] = make([]float64, photoWidth)
		for x := range photoWidth {
			u := float64(x) / photoWidth
			h := 0.0
			for k := range 3 {
				h += math.Sin(u*rd.freqs[k]*math.Pi+rd.phases[k]) / float64(k*k+1)
			}
			heights[i][x] = rd.base - rd.amplitude*h
		}
	}
	for y := range photoHeight {
		v := float64(y) / photoHeight
		for x := range photoWidth {
			// The nearest ridge covering the pixel decides its color.
			c := mix(top, horizon, math.Min(v/0.6, 1))
			for i := len(ridges) - 1; i >= 0; i-- {
				if v >= heights[i][x] {
					c = ridges[i].color
					depthBelow := v - heights[i][x]
					if ridges[i].snow && depthBelow < 0.04 {
						c = mix([3]float64{245, 248, 252}, c, depthBelow/0.04)
					}
					// Darker further down, as if in shadow.
					c = mix(c, [3]float64{c[0] * 0.7, c[1] * 0.7, c[2] * 0.7}, math.Min(depthBelow*3, 1))
					break
				}
			}
			o := img.PixOffset(x, y)
			img.Pix[o], img.Pix[o+1], img.Pix[o+2], img.Pix[o+3] = uint8(c[0]), uint8(c[1]), uint8(c[2]), 255
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// mix blends from a to b by f, from 0 to 1.
func mix(a, b [3]float64, f float64) [3]float64 {
	f = math.Min(math.Max(f, 0), 1)
	return [3]float64{a[0] + (b[0]-a[0])*f, a[1] + (b[1]-a[1])*f, a[2] + (b[2]-a[2])*f}
}
