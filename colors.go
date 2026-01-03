package fansiterm

import (
	"image"
	"image/color"
)

// Color wraps color.Color and implements image.Image and color.Model.
//
// Its main purpose is to avoid repeatedly instantiating image.Uniform
// when drawing with solid colors. This allows Color to be used directly
// anywhere color.Color, image.Image, or color.Model are expected
type Color struct {
	color.Color
}

// NewOpaqueColor returns a Color with full opacity (alpha = 255).
func NewOpaqueColor(r, g, b uint8) Color {
	return Color{color.RGBA{r, g, b, 255}}
}

// NewColor returns a Color with the specified RGBA values.
func NewColor(r, g, b, a uint8) Color {
	return Color{color.RGBA{r, g, b, a}}
}

// At implements image.Image by returning the embedded color value.
func (c Color) At(int, int) color.Color {
	return c.Color
}

// Bounds implements image.Image.
// It returns an extremely large bounding rectangle to satisfy
// the image.Image interface when Color is used as an image.
func (c Color) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}

// ColorModel implements image.Image and color.Model.
// Returns itself to satisfy both interfaces.
func (c Color) ColorModel() color.Model {
	return c
}

// Convert implements color.Model.
// Since this Color is already assumed to be in the correct native format,
// Convert is a stub and returns the input as-is.
func (c Color) Convert(cin color.Color) color.Color {
	return cin
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
