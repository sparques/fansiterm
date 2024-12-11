package fansiterm

import (
	"image"
	"image/color"
)

// Color both implements color.Color and image.Image.
// image.Image needs a color.Model, so for convenience's
// sake, Color also implements color.Model so it can
// simply have ColorModel() return itself.
// The main purpose of Color is so there is no need to
// instantiate an image.Unform everytime we need to
// draw something in a particular color.
type Color struct {
	color.Color
}

// NewOpaqueColor returns a Color that has a fully opaque alpha value.
func NewOpaqueColor(r, g, b uint8) Color {
	return Color{color.RGBA{r, g, b, 255}}
}

// NewColor returns a new Color.
func NewColor(r, g, b, a uint8) Color {
	return Color{color.RGBA{r, g, b, a}}
}

// At implements image.Image
func (c Color) At(int, int) color.Color {
	return c.Color
}

// Bounds implements image.Image
func (c Color) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}

// ColorModel implements image.Image
func (c Color) ColorModel() color.Model {
	return c
}

func (c Color) Convert(cin color.Color) color.Color {
	return cin
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

type colorSystem struct {
	model       color.Model
	PaletteANSI [16]Color
	Palette256  [256]Color
}

// Use the color model from the backing image, regardles of where it
// comes from (allocated here or provided)
func NewColorSystem(m color.Model) *colorSystem {
	cs := &colorSystem{model: m}

	// init PaletteANSI
	cs.PaletteANSI = [16]Color{
		cs.NewRGB(0, 0, 0),
		cs.NewRGB(127, 0, 0),
		cs.NewRGB(0, 170, 0),
		cs.NewRGB(170, 85, 0),
		cs.NewRGB(0, 0, 170),
		cs.NewRGB(170, 0, 170),
		cs.NewRGB(0, 170, 170),
		cs.NewRGB(200, 200, 200),
		cs.NewRGB(85, 85, 85),
		cs.NewRGB(255, 0, 0),
		cs.NewRGB(85, 255, 85),
		cs.NewRGB(255, 255, 85),
		cs.NewRGB(85, 85, 255),
		cs.NewRGB(255, 85, 255),
		cs.NewRGB(85, 255, 255),
		cs.NewRGB(255, 255, 255),
	}
	// init Palette256
	cs.Palette256 = [256]Color{
		cs.NewRGB(0, 0, 0),
		cs.NewRGB(128, 0, 0),
		cs.NewRGB(0, 128, 0),
		cs.NewRGB(128, 128, 0),
		cs.NewRGB(0, 0, 128),
		cs.NewRGB(128, 0, 128),
		cs.NewRGB(0, 128, 128),
		cs.NewRGB(192, 192, 192),
		cs.NewRGB(128, 128, 128),
		cs.NewRGB(255, 0, 0),
		cs.NewRGB(0, 255, 0),
		cs.NewRGB(255, 255, 0),
		cs.NewRGB(0, 0, 255),
		cs.NewRGB(255, 0, 255),
		cs.NewRGB(0, 255, 255),
		cs.NewRGB(255, 255, 255),
		cs.NewRGB(0, 0, 0),
		cs.NewRGB(0, 0, 95),
		cs.NewRGB(0, 0, 135),
		cs.NewRGB(0, 0, 175),
		cs.NewRGB(0, 0, 215),
		cs.NewRGB(0, 0, 255),
		cs.NewRGB(0, 95, 0),
		cs.NewRGB(0, 95, 95),
		cs.NewRGB(0, 95, 135),
		cs.NewRGB(0, 95, 175),
		cs.NewRGB(0, 95, 215),
		cs.NewRGB(0, 95, 255),
		cs.NewRGB(0, 135, 0),
		cs.NewRGB(0, 135, 95),
		cs.NewRGB(0, 135, 135),
		cs.NewRGB(0, 135, 175),
		cs.NewRGB(0, 135, 215),
		cs.NewRGB(0, 135, 255),
		cs.NewRGB(0, 175, 0),
		cs.NewRGB(0, 175, 95),
		cs.NewRGB(0, 175, 135),
		cs.NewRGB(0, 175, 175),
		cs.NewRGB(0, 175, 215),
		cs.NewRGB(0, 175, 255),
		cs.NewRGB(0, 215, 0),
		cs.NewRGB(0, 215, 95),
		cs.NewRGB(0, 215, 135),
		cs.NewRGB(0, 215, 175),
		cs.NewRGB(0, 215, 215),
		cs.NewRGB(0, 215, 255),
		cs.NewRGB(0, 255, 0),
		cs.NewRGB(0, 255, 95),
		cs.NewRGB(0, 255, 135),
		cs.NewRGB(0, 255, 175),
		cs.NewRGB(0, 255, 215),
		cs.NewRGB(0, 255, 255),
		cs.NewRGB(95, 0, 0),
		cs.NewRGB(95, 0, 95),
		cs.NewRGB(95, 0, 135),
		cs.NewRGB(95, 0, 175),
		cs.NewRGB(95, 0, 215),
		cs.NewRGB(95, 0, 255),
		cs.NewRGB(95, 95, 0),
		cs.NewRGB(95, 95, 95),
		cs.NewRGB(95, 95, 135),
		cs.NewRGB(95, 95, 175),
		cs.NewRGB(95, 95, 215),
		cs.NewRGB(95, 95, 255),
		cs.NewRGB(95, 135, 0),
		cs.NewRGB(95, 135, 95),
		cs.NewRGB(95, 135, 135),
		cs.NewRGB(95, 135, 175),
		cs.NewRGB(95, 135, 215),
		cs.NewRGB(95, 135, 255),
		cs.NewRGB(95, 175, 0),
		cs.NewRGB(95, 175, 95),
		cs.NewRGB(95, 175, 135),
		cs.NewRGB(95, 175, 175),
		cs.NewRGB(95, 175, 215),
		cs.NewRGB(95, 175, 255),
		cs.NewRGB(95, 215, 0),
		cs.NewRGB(95, 215, 95),
		cs.NewRGB(95, 215, 135),
		cs.NewRGB(95, 215, 175),
		cs.NewRGB(95, 215, 215),
		cs.NewRGB(95, 215, 255),
		cs.NewRGB(95, 255, 0),
		cs.NewRGB(95, 255, 95),
		cs.NewRGB(95, 255, 135),
		cs.NewRGB(95, 255, 175),
		cs.NewRGB(95, 255, 215),
		cs.NewRGB(95, 255, 255),
		cs.NewRGB(135, 0, 0),
		cs.NewRGB(135, 0, 95),
		cs.NewRGB(135, 0, 135),
		cs.NewRGB(135, 0, 175),
		cs.NewRGB(135, 0, 215),
		cs.NewRGB(135, 0, 255),
		cs.NewRGB(135, 95, 0),
		cs.NewRGB(135, 95, 95),
		cs.NewRGB(135, 95, 135),
		cs.NewRGB(135, 95, 175),
		cs.NewRGB(135, 95, 215),
		cs.NewRGB(135, 95, 255),
		cs.NewRGB(135, 135, 0),
		cs.NewRGB(135, 135, 95),
		cs.NewRGB(135, 135, 135),
		cs.NewRGB(135, 135, 175),
		cs.NewRGB(135, 135, 215),
		cs.NewRGB(135, 135, 255),
		cs.NewRGB(135, 175, 0),
		cs.NewRGB(135, 175, 95),
		cs.NewRGB(135, 175, 135),
		cs.NewRGB(135, 175, 175),
		cs.NewRGB(135, 175, 215),
		cs.NewRGB(135, 175, 255),
		cs.NewRGB(135, 215, 0),
		cs.NewRGB(135, 215, 95),
		cs.NewRGB(135, 215, 135),
		cs.NewRGB(135, 215, 175),
		cs.NewRGB(135, 215, 215),
		cs.NewRGB(135, 215, 255),
		cs.NewRGB(135, 255, 0),
		cs.NewRGB(135, 255, 95),
		cs.NewRGB(135, 255, 135),
		cs.NewRGB(135, 255, 175),
		cs.NewRGB(135, 255, 215),
		cs.NewRGB(135, 255, 255),
		cs.NewRGB(175, 0, 0),
		cs.NewRGB(175, 0, 95),
		cs.NewRGB(175, 0, 135),
		cs.NewRGB(175, 0, 175),
		cs.NewRGB(175, 0, 215),
		cs.NewRGB(175, 0, 255),
		cs.NewRGB(175, 95, 0),
		cs.NewRGB(175, 95, 95),
		cs.NewRGB(175, 95, 135),
		cs.NewRGB(175, 95, 175),
		cs.NewRGB(175, 95, 215),
		cs.NewRGB(175, 95, 255),
		cs.NewRGB(175, 135, 0),
		cs.NewRGB(175, 135, 95),
		cs.NewRGB(175, 135, 135),
		cs.NewRGB(175, 135, 175),
		cs.NewRGB(175, 135, 215),
		cs.NewRGB(175, 135, 255),
		cs.NewRGB(175, 175, 0),
		cs.NewRGB(175, 175, 95),
		cs.NewRGB(175, 175, 135),
		cs.NewRGB(175, 175, 175),
		cs.NewRGB(175, 175, 215),
		cs.NewRGB(175, 175, 255),
		cs.NewRGB(175, 215, 0),
		cs.NewRGB(175, 215, 95),
		cs.NewRGB(175, 215, 135),
		cs.NewRGB(175, 215, 175),
		cs.NewRGB(175, 215, 215),
		cs.NewRGB(175, 215, 255),
		cs.NewRGB(175, 255, 0),
		cs.NewRGB(175, 255, 95),
		cs.NewRGB(175, 255, 135),
		cs.NewRGB(175, 255, 175),
		cs.NewRGB(175, 255, 215),
		cs.NewRGB(175, 255, 255),
		cs.NewRGB(215, 0, 0),
		cs.NewRGB(215, 0, 95),
		cs.NewRGB(215, 0, 135),
		cs.NewRGB(215, 0, 175),
		cs.NewRGB(215, 0, 215),
		cs.NewRGB(215, 0, 255),
		cs.NewRGB(215, 95, 0),
		cs.NewRGB(215, 95, 95),
		cs.NewRGB(215, 95, 135),
		cs.NewRGB(215, 95, 175),
		cs.NewRGB(215, 95, 215),
		cs.NewRGB(215, 95, 255),
		cs.NewRGB(215, 135, 0),
		cs.NewRGB(215, 135, 95),
		cs.NewRGB(215, 135, 135),
		cs.NewRGB(215, 135, 175),
		cs.NewRGB(215, 135, 215),
		cs.NewRGB(215, 135, 255),
		cs.NewRGB(215, 175, 0),
		cs.NewRGB(215, 175, 95),
		cs.NewRGB(215, 175, 135),
		cs.NewRGB(215, 175, 175),
		cs.NewRGB(215, 175, 215),
		cs.NewRGB(215, 175, 255),
		cs.NewRGB(215, 215, 0),
		cs.NewRGB(215, 215, 95),
		cs.NewRGB(215, 215, 135),
		cs.NewRGB(215, 215, 175),
		cs.NewRGB(215, 215, 215),
		cs.NewRGB(215, 215, 255),
		cs.NewRGB(215, 255, 0),
		cs.NewRGB(215, 255, 95),
		cs.NewRGB(215, 255, 135),
		cs.NewRGB(215, 255, 175),
		cs.NewRGB(215, 255, 215),
		cs.NewRGB(215, 255, 255),
		cs.NewRGB(255, 0, 0),
		cs.NewRGB(255, 0, 95),
		cs.NewRGB(255, 0, 135),
		cs.NewRGB(255, 0, 175),
		cs.NewRGB(255, 0, 215),
		cs.NewRGB(255, 0, 255),
		cs.NewRGB(255, 95, 0),
		cs.NewRGB(255, 95, 95),
		cs.NewRGB(255, 95, 135),
		cs.NewRGB(255, 95, 175),
		cs.NewRGB(255, 95, 215),
		cs.NewRGB(255, 95, 255),
		cs.NewRGB(255, 135, 0),
		cs.NewRGB(255, 135, 95),
		cs.NewRGB(255, 135, 135),
		cs.NewRGB(255, 135, 175),
		cs.NewRGB(255, 135, 215),
		cs.NewRGB(255, 135, 255),
		cs.NewRGB(255, 175, 0),
		cs.NewRGB(255, 175, 95),
		cs.NewRGB(255, 175, 135),
		cs.NewRGB(255, 175, 175),
		cs.NewRGB(255, 175, 215),
		cs.NewRGB(255, 175, 255),
		cs.NewRGB(255, 215, 0),
		cs.NewRGB(255, 215, 95),
		cs.NewRGB(255, 215, 135),
		cs.NewRGB(255, 215, 175),
		cs.NewRGB(255, 215, 215),
		cs.NewRGB(255, 215, 255),
		cs.NewRGB(255, 255, 0),
		cs.NewRGB(255, 255, 95),
		cs.NewRGB(255, 255, 135),
		cs.NewRGB(255, 255, 175),
		cs.NewRGB(255, 255, 215),
		cs.NewRGB(255, 255, 255),
		cs.NewRGB(8, 8, 8),
		cs.NewRGB(18, 18, 18),
		cs.NewRGB(28, 28, 28),
		cs.NewRGB(38, 38, 38),
		cs.NewRGB(48, 48, 48),
		cs.NewRGB(58, 58, 58),
		cs.NewRGB(68, 68, 68),
		cs.NewRGB(78, 78, 78),
		cs.NewRGB(88, 88, 88),
		cs.NewRGB(98, 98, 98),
		cs.NewRGB(108, 108, 108),
		cs.NewRGB(118, 118, 118),
		cs.NewRGB(128, 128, 128),
		cs.NewRGB(138, 138, 138),
		cs.NewRGB(148, 148, 148),
		cs.NewRGB(158, 158, 158),
		cs.NewRGB(168, 168, 168),
		cs.NewRGB(178, 178, 178),
		cs.NewRGB(188, 188, 188),
		cs.NewRGB(198, 198, 198),
		cs.NewRGB(208, 208, 208),
		cs.NewRGB(218, 218, 218),
		cs.NewRGB(228, 228, 228),
		cs.NewRGB(238, 238, 238),
	}
	return cs
}

func (cs *colorSystem) NewRGB(r, g, b uint8) Color {
	return Color{cs.model.Convert(color.RGBA{r, g, b, 255})}
}
