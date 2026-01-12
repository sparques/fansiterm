//go:build color_mono && !color_ansi && !color_256

package fansiterm

import (
	"image"
	"image/color"

	"github.com/sparques/fansiterm/tiles"
)

type Color struct {
	tiles.Mono
}

var (
	pixelOn  = Color{tiles.Mono(true)}
	pixelOff = Color{tiles.Mono(false)}
)

func init() {
	defaultFg = pixelOn
	defaultBg = pixelOff
}

// NewOpaqueColor returns a Color with full opacity (alpha = 255).
func NewOpaqueColor(r, g, b uint8) Color {
	if max(r, g, b) > 127 {
		return Color{tiles.Mono(true)}
	}
	return Color{tiles.Mono(false)}
}

// NewColor returns a Color with the specified RGBA values.
func NewColor(r, g, b, a uint8) Color {
	if max(r, g, b) > 127 {
		return Color{tiles.Mono(true)}
	}
	return Color{tiles.Mono(false)}
}

// ColorModel implements image.Image and color.Model.
func (c Color) ColorModel() color.Model {
	return tiles.MonoModel
}

// At implements image.Image by returning the embedded color value.
func (c Color) At(int, int) color.Color {
	return c.Mono
}

// Convert implements color.Model.
func (c Color) Convert(cin color.Color) color.Color {
	return tiles.MonoModel.Convert(cin)
}

// Bounds implements image.Image.
// It returns an extremely large bounding rectangle to satisfy
// the image.Image interface when Color is used as an image.
func (c Color) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}
