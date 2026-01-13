package fansiterm

import (
	"image"
	"image/color"
)

// Color wraps color.Color and implements image.Image and color.Model.
//
// It's defined by a particular implementation that is specified with
// a build tag.
//
// Its main purpose is to avoid repeatedly instantiating image.Uniform
// when drawing with solid colors. This allows Color to be used directly
// anywhere color.Color, image.Image, or color.Model are expected

// Ensure at build-time the Color implementation has implemented both
// color.Color and image.Image.
var (
	_ color.Color = Color{}
	_ image.Image = Color{}

	PaletteANSI []Color
	Palette256  []Color

	defaultFg, defaultBg Color
)

func ColorANSI(n int) (c Color) {
	if len(PaletteANSI) == 0 {
		if n == 0 || n == 8 {
			return defaultBg
		} else {
			return defaultFg
		}
	}
	return PaletteANSI[n%16]
}

func Color256(n int) (c Color) {
	if len(Palette256) == 0 {
		// Add more to this mapping?
		if n == 0 || n == 8 {
			return defaultBg
		} else {
			return defaultFg
		}
	}
	return Palette256[n%256]
}

func NewColorFromRGBA(c color.RGBA) Color {
	return NewOpaqueColor(c.R, c.G, c.B)
}

// Colorizer is an adapter that allows a pixel.Color's RGBA method
// to satisfy the color.Color interface used by the Go image packages.
//
// Example usage:
//
//	pixelColor := pixel.NewColor[pixel.RGB888](127,127,127)
//	drawImage.Set(x, y, Colorizer(pixelColor.RGBA))
type Colorizer func() color.RGBA

// RGBA implements color.Color by invoking the wrapped function.
func (c Colorizer) RGBA() (r, g, b, a uint32) {
	v := c()
	return uint32(v.R), uint32(v.G), uint32(v.B), uint32(v.A)
}
