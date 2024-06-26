package fansiterm

import (
	"image"
	"image/color"
)

var (
	// These Colors are for the 4-bit ANSI colors
	// Since they're exported, they can be overridden.
	// It would be convient to have a pallet, but given
	// TrueColor support, why bother?
	ColorBlack         = NewOpaqueColor(0, 0, 0)
	ColorBrightBlack   = NewOpaqueColor(85, 85, 85)
	ColorRed           = NewOpaqueColor(127, 0, 0)
	ColorBrightRed     = NewOpaqueColor(255, 0, 0)
	ColorGreen         = NewOpaqueColor(0, 170, 0)
	ColorBrightGreen   = NewOpaqueColor(85, 255, 85)
	ColorYellow        = NewOpaqueColor(170, 85, 0)
	ColorBrightYellow  = NewOpaqueColor(255, 255, 85)
	ColorBlue          = NewOpaqueColor(0, 0, 170)
	ColorBrightBlue    = NewOpaqueColor(85, 85, 255)
	ColorMagenta       = NewOpaqueColor(170, 0, 170)
	ColorBrightMagenta = NewOpaqueColor(255, 85, 255)
	ColorCyan          = NewOpaqueColor(0, 170, 170)
	ColorBrightCyan    = NewOpaqueColor(85, 255, 255)
	// Okay, I deviated from VGA colors here. VGA "white" is way too gray.
	ColorWhite = NewOpaqueColor(240, 240, 240)
	// ColorWhite       = NewOpaqueColor(170, 170, 170)
	ColorBrightWhite = NewOpaqueColor(255, 255, 255)
)

// Colors256 defines the default set of 256 Colors
var Colors256 = [256]Color{
	NewOpaqueColor(0, 0, 0),
	NewOpaqueColor(128, 0, 0),
	NewOpaqueColor(0, 128, 0),
	NewOpaqueColor(128, 128, 0),
	NewOpaqueColor(0, 0, 128),
	NewOpaqueColor(128, 0, 128),
	NewOpaqueColor(0, 128, 128),
	NewOpaqueColor(192, 192, 192),
	NewOpaqueColor(128, 128, 128),
	NewOpaqueColor(255, 0, 0),
	NewOpaqueColor(0, 255, 0),
	NewOpaqueColor(255, 255, 0),
	NewOpaqueColor(0, 0, 255),
	NewOpaqueColor(255, 0, 255),
	NewOpaqueColor(0, 255, 255),
	NewOpaqueColor(255, 255, 255),
	NewOpaqueColor(0, 0, 0),
	NewOpaqueColor(0, 0, 95),
	NewOpaqueColor(0, 0, 135),
	NewOpaqueColor(0, 0, 175),
	NewOpaqueColor(0, 0, 215),
	NewOpaqueColor(0, 0, 255),
	NewOpaqueColor(0, 95, 0),
	NewOpaqueColor(0, 95, 95),
	NewOpaqueColor(0, 95, 135),
	NewOpaqueColor(0, 95, 175),
	NewOpaqueColor(0, 95, 215),
	NewOpaqueColor(0, 95, 255),
	NewOpaqueColor(0, 135, 0),
	NewOpaqueColor(0, 135, 95),
	NewOpaqueColor(0, 135, 135),
	NewOpaqueColor(0, 135, 175),
	NewOpaqueColor(0, 135, 215),
	NewOpaqueColor(0, 135, 255),
	NewOpaqueColor(0, 175, 0),
	NewOpaqueColor(0, 175, 95),
	NewOpaqueColor(0, 175, 135),
	NewOpaqueColor(0, 175, 175),
	NewOpaqueColor(0, 175, 215),
	NewOpaqueColor(0, 175, 255),
	NewOpaqueColor(0, 215, 0),
	NewOpaqueColor(0, 215, 95),
	NewOpaqueColor(0, 215, 135),
	NewOpaqueColor(0, 215, 175),
	NewOpaqueColor(0, 215, 215),
	NewOpaqueColor(0, 215, 255),
	NewOpaqueColor(0, 255, 0),
	NewOpaqueColor(0, 255, 95),
	NewOpaqueColor(0, 255, 135),
	NewOpaqueColor(0, 255, 175),
	NewOpaqueColor(0, 255, 215),
	NewOpaqueColor(0, 255, 255),
	NewOpaqueColor(95, 0, 0),
	NewOpaqueColor(95, 0, 95),
	NewOpaqueColor(95, 0, 135),
	NewOpaqueColor(95, 0, 175),
	NewOpaqueColor(95, 0, 215),
	NewOpaqueColor(95, 0, 255),
	NewOpaqueColor(95, 95, 0),
	NewOpaqueColor(95, 95, 95),
	NewOpaqueColor(95, 95, 135),
	NewOpaqueColor(95, 95, 175),
	NewOpaqueColor(95, 95, 215),
	NewOpaqueColor(95, 95, 255),
	NewOpaqueColor(95, 135, 0),
	NewOpaqueColor(95, 135, 95),
	NewOpaqueColor(95, 135, 135),
	NewOpaqueColor(95, 135, 175),
	NewOpaqueColor(95, 135, 215),
	NewOpaqueColor(95, 135, 255),
	NewOpaqueColor(95, 175, 0),
	NewOpaqueColor(95, 175, 95),
	NewOpaqueColor(95, 175, 135),
	NewOpaqueColor(95, 175, 175),
	NewOpaqueColor(95, 175, 215),
	NewOpaqueColor(95, 175, 255),
	NewOpaqueColor(95, 215, 0),
	NewOpaqueColor(95, 215, 95),
	NewOpaqueColor(95, 215, 135),
	NewOpaqueColor(95, 215, 175),
	NewOpaqueColor(95, 215, 215),
	NewOpaqueColor(95, 215, 255),
	NewOpaqueColor(95, 255, 0),
	NewOpaqueColor(95, 255, 95),
	NewOpaqueColor(95, 255, 135),
	NewOpaqueColor(95, 255, 175),
	NewOpaqueColor(95, 255, 215),
	NewOpaqueColor(95, 255, 255),
	NewOpaqueColor(135, 0, 0),
	NewOpaqueColor(135, 0, 95),
	NewOpaqueColor(135, 0, 135),
	NewOpaqueColor(135, 0, 175),
	NewOpaqueColor(135, 0, 215),
	NewOpaqueColor(135, 0, 255),
	NewOpaqueColor(135, 95, 0),
	NewOpaqueColor(135, 95, 95),
	NewOpaqueColor(135, 95, 135),
	NewOpaqueColor(135, 95, 175),
	NewOpaqueColor(135, 95, 215),
	NewOpaqueColor(135, 95, 255),
	NewOpaqueColor(135, 135, 0),
	NewOpaqueColor(135, 135, 95),
	NewOpaqueColor(135, 135, 135),
	NewOpaqueColor(135, 135, 175),
	NewOpaqueColor(135, 135, 215),
	NewOpaqueColor(135, 135, 255),
	NewOpaqueColor(135, 175, 0),
	NewOpaqueColor(135, 175, 95),
	NewOpaqueColor(135, 175, 135),
	NewOpaqueColor(135, 175, 175),
	NewOpaqueColor(135, 175, 215),
	NewOpaqueColor(135, 175, 255),
	NewOpaqueColor(135, 215, 0),
	NewOpaqueColor(135, 215, 95),
	NewOpaqueColor(135, 215, 135),
	NewOpaqueColor(135, 215, 175),
	NewOpaqueColor(135, 215, 215),
	NewOpaqueColor(135, 215, 255),
	NewOpaqueColor(135, 255, 0),
	NewOpaqueColor(135, 255, 95),
	NewOpaqueColor(135, 255, 135),
	NewOpaqueColor(135, 255, 175),
	NewOpaqueColor(135, 255, 215),
	NewOpaqueColor(135, 255, 255),
	NewOpaqueColor(175, 0, 0),
	NewOpaqueColor(175, 0, 95),
	NewOpaqueColor(175, 0, 135),
	NewOpaqueColor(175, 0, 175),
	NewOpaqueColor(175, 0, 215),
	NewOpaqueColor(175, 0, 255),
	NewOpaqueColor(175, 95, 0),
	NewOpaqueColor(175, 95, 95),
	NewOpaqueColor(175, 95, 135),
	NewOpaqueColor(175, 95, 175),
	NewOpaqueColor(175, 95, 215),
	NewOpaqueColor(175, 95, 255),
	NewOpaqueColor(175, 135, 0),
	NewOpaqueColor(175, 135, 95),
	NewOpaqueColor(175, 135, 135),
	NewOpaqueColor(175, 135, 175),
	NewOpaqueColor(175, 135, 215),
	NewOpaqueColor(175, 135, 255),
	NewOpaqueColor(175, 175, 0),
	NewOpaqueColor(175, 175, 95),
	NewOpaqueColor(175, 175, 135),
	NewOpaqueColor(175, 175, 175),
	NewOpaqueColor(175, 175, 215),
	NewOpaqueColor(175, 175, 255),
	NewOpaqueColor(175, 215, 0),
	NewOpaqueColor(175, 215, 95),
	NewOpaqueColor(175, 215, 135),
	NewOpaqueColor(175, 215, 175),
	NewOpaqueColor(175, 215, 215),
	NewOpaqueColor(175, 215, 255),
	NewOpaqueColor(175, 255, 0),
	NewOpaqueColor(175, 255, 95),
	NewOpaqueColor(175, 255, 135),
	NewOpaqueColor(175, 255, 175),
	NewOpaqueColor(175, 255, 215),
	NewOpaqueColor(175, 255, 255),
	NewOpaqueColor(215, 0, 0),
	NewOpaqueColor(215, 0, 95),
	NewOpaqueColor(215, 0, 135),
	NewOpaqueColor(215, 0, 175),
	NewOpaqueColor(215, 0, 215),
	NewOpaqueColor(215, 0, 255),
	NewOpaqueColor(215, 95, 0),
	NewOpaqueColor(215, 95, 95),
	NewOpaqueColor(215, 95, 135),
	NewOpaqueColor(215, 95, 175),
	NewOpaqueColor(215, 95, 215),
	NewOpaqueColor(215, 95, 255),
	NewOpaqueColor(215, 135, 0),
	NewOpaqueColor(215, 135, 95),
	NewOpaqueColor(215, 135, 135),
	NewOpaqueColor(215, 135, 175),
	NewOpaqueColor(215, 135, 215),
	NewOpaqueColor(215, 135, 255),
	NewOpaqueColor(215, 175, 0),
	NewOpaqueColor(215, 175, 95),
	NewOpaqueColor(215, 175, 135),
	NewOpaqueColor(215, 175, 175),
	NewOpaqueColor(215, 175, 215),
	NewOpaqueColor(215, 175, 255),
	NewOpaqueColor(215, 215, 0),
	NewOpaqueColor(215, 215, 95),
	NewOpaqueColor(215, 215, 135),
	NewOpaqueColor(215, 215, 175),
	NewOpaqueColor(215, 215, 215),
	NewOpaqueColor(215, 215, 255),
	NewOpaqueColor(215, 255, 0),
	NewOpaqueColor(215, 255, 95),
	NewOpaqueColor(215, 255, 135),
	NewOpaqueColor(215, 255, 175),
	NewOpaqueColor(215, 255, 215),
	NewOpaqueColor(215, 255, 255),
	NewOpaqueColor(255, 0, 0),
	NewOpaqueColor(255, 0, 95),
	NewOpaqueColor(255, 0, 135),
	NewOpaqueColor(255, 0, 175),
	NewOpaqueColor(255, 0, 215),
	NewOpaqueColor(255, 0, 255),
	NewOpaqueColor(255, 95, 0),
	NewOpaqueColor(255, 95, 95),
	NewOpaqueColor(255, 95, 135),
	NewOpaqueColor(255, 95, 175),
	NewOpaqueColor(255, 95, 215),
	NewOpaqueColor(255, 95, 255),
	NewOpaqueColor(255, 135, 0),
	NewOpaqueColor(255, 135, 95),
	NewOpaqueColor(255, 135, 135),
	NewOpaqueColor(255, 135, 175),
	NewOpaqueColor(255, 135, 215),
	NewOpaqueColor(255, 135, 255),
	NewOpaqueColor(255, 175, 0),
	NewOpaqueColor(255, 175, 95),
	NewOpaqueColor(255, 175, 135),
	NewOpaqueColor(255, 175, 175),
	NewOpaqueColor(255, 175, 215),
	NewOpaqueColor(255, 175, 255),
	NewOpaqueColor(255, 215, 0),
	NewOpaqueColor(255, 215, 95),
	NewOpaqueColor(255, 215, 135),
	NewOpaqueColor(255, 215, 175),
	NewOpaqueColor(255, 215, 215),
	NewOpaqueColor(255, 215, 255),
	NewOpaqueColor(255, 255, 0),
	NewOpaqueColor(255, 255, 95),
	NewOpaqueColor(255, 255, 135),
	NewOpaqueColor(255, 255, 175),
	NewOpaqueColor(255, 255, 215),
	NewOpaqueColor(255, 255, 255),
	NewOpaqueColor(8, 8, 8),
	NewOpaqueColor(18, 18, 18),
	NewOpaqueColor(28, 28, 28),
	NewOpaqueColor(38, 38, 38),
	NewOpaqueColor(48, 48, 48),
	NewOpaqueColor(58, 58, 58),
	NewOpaqueColor(68, 68, 68),
	NewOpaqueColor(78, 78, 78),
	NewOpaqueColor(88, 88, 88),
	NewOpaqueColor(98, 98, 98),
	NewOpaqueColor(108, 108, 108),
	NewOpaqueColor(118, 118, 118),
	NewOpaqueColor(128, 128, 128),
	NewOpaqueColor(138, 138, 138),
	NewOpaqueColor(148, 148, 148),
	NewOpaqueColor(158, 158, 158),
	NewOpaqueColor(168, 168, 168),
	NewOpaqueColor(178, 178, 178),
	NewOpaqueColor(188, 188, 188),
	NewOpaqueColor(198, 198, 198),
	NewOpaqueColor(208, 208, 208),
	NewOpaqueColor(218, 218, 218),
	NewOpaqueColor(228, 228, 228),
	NewOpaqueColor(238, 238, 238),
}

// Color both implements color.Color and image.Image.
// image.Image needs a color.Model, so for convenience's
// sake, Color also implements color.Model so it can
// simply have ColorModel() return itself.
// The main purpose of Color is so there is no need to
// instantiate an image.Unform everytime we need to
// draw something in a particular color.
type Color struct {
	rgba color.RGBA
}

// NewOpaqueColor returns a Color that has a fully opaque alpha value.
func NewOpaqueColor(r, g, b uint8) Color {
	return Color{color.RGBA{r, g, b, 255}}
}

// NewColor returns a new Color.
func NewColor(r, g, b, a uint8) Color {
	return Color{color.RGBA{r, g, b, a}}
}

// RGBA implements color.Color
func (c Color) RGBA() (r, g, b, a uint32) {
	return c.rgba.RGBA()
}

// At implements image.Image
func (c Color) At(int, int) color.Color {
	return c.rgba
}

// Bounds implements image.Image
func (c Color) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}

// ColorModel implements image.Image
func (c Color) ColorModel() color.Model {
	return c
}

// Convert (fake) implements color.Model.
func (c Color) Convert(c2 color.Color) color.Color {
	return c2
}

// the tinygo.org/x/drivers/pixel package has a somewhat incompatible
// color interface with the color.Color interface. This type definition
// and it's associated function allows a pixel.Color's RGBA method to be
// cast so that it implements the color.Color interface.
// Example:
// pixelColor := pixel.NewColor[pixel.RGB888](127,127,127)
// drawImage.Set(xPos,yPos, Colorizer(pixelColor.RGBA))
type Colorizer func() color.RGBA

func (c Colorizer) RGBA() (r, g, b, a uint32) {
	v := c()
	return uint32(v.R), uint32(v.G), uint32(v.B), uint32(v.A)
}
