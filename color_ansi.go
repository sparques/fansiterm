//go:build !color_mono

package fansiterm

import (
	"image"
	"image/color"
)

type Color struct {
	RGB color.RGBA
}

func init() {
	PaletteANSI = []Color{
		NewOpaqueColor(0, 0, 0),
		NewOpaqueColor(127, 0, 0),
		NewOpaqueColor(0, 170, 0),
		NewOpaqueColor(170, 85, 0),
		NewOpaqueColor(0, 0, 170),
		NewOpaqueColor(170, 0, 170),
		NewOpaqueColor(0, 170, 170),
		NewOpaqueColor(200, 200, 200),
		NewOpaqueColor(85, 85, 85),
		NewOpaqueColor(255, 0, 0),
		NewOpaqueColor(85, 255, 85),
		NewOpaqueColor(255, 255, 85),
		NewOpaqueColor(85, 85, 255),
		NewOpaqueColor(255, 85, 255),
		NewOpaqueColor(85, 255, 255),
		NewOpaqueColor(255, 255, 255),
	}
	defaultFg = PaletteANSI[7]
	defaultBg = PaletteANSI[0]
}

// NewOpaqueColor returns a Color with full opacity (alpha = 255).
func NewOpaqueColor(r, g, b uint8) Color {
	return Color{color.RGBA{r, g, b, 255}}
}

// NewColor returns a Color with the specified RGBA values.
func NewColor(r, g, b, a uint8) Color {
	return Color{color.RGBA{r, g, b, a}}
}

func (c Color) RGBA() (r, g, b, a uint32) {
	return c.RGB.RGBA()
}

// ColorModel implements image.Image and color.Model.
func (c Color) ColorModel() color.Model {
	return color.RGBAModel
}

// At implements image.Image by returning the embedded color value.
func (c Color) At(int, int) color.Color {
	return c
}

// Convert implements color.Model.
func (c Color) Convert(cin color.Color) color.Color {
	return color.RGBAModel.Convert(cin)
}

// Bounds implements image.Image.
// It returns an extremely large bounding rectangle to satisfy
// the image.Image interface when Color is used as an image.
func (c Color) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}
