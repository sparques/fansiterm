package fansiterm

import (
	"image"
	"image/color"
)

type RGBColor struct {
	R, G, B uint8
}

func (c RGBColor) RGBA() (r, g, b, a uint32) {
	r = uint32(c.R) * 0x101
	g = uint32(c.G) * 0x101
	b = uint32(c.B) * 0x101
	a = 255 * 0x101
	return
}

func rgbColorModel(c color.Color) color.Color {
	if native, ok := c.(RGBColor); ok {
		return native
	}
	r, g, b, _ := c.RGBA()
	return RGBColor{
		uint8(r / 0x101),
		uint8(g / 0x101),
		uint8(b / 0x101),
	}
}

type RGBImage struct {
	Pix []uint8
	image.Rectangle
}

func NewRGBImage(r image.Rectangle) *RGBImage {
	return &RGBImage{
		Rectangle: r,
		Pix:       make([]uint8, r.Dx()*r.Dy()*3),
	}
}

func (p *RGBImage) At(x, y int) color.Color {
	i := y*p.Dx()*3 + x*3
	return RGBColor{p.Pix[i], p.Pix[i+1], p.Pix[i+2]}
}

func (p *RGBImage) Set(x, y int, c color.Color) {
	nc := rgbColorModel(c).(RGBColor)
	i := y*p.Dx()*3 + x*3
	p.Pix[i] = nc.R
	p.Pix[i+1] = nc.G
	p.Pix[i+2] = nc.B
}

func (p *RGBImage) Bounds() image.Rectangle {
	return p.Rectangle
}

func (p *RGBImage) ColorModel() color.Model {
	return color.ModelFunc(rgbColorModel)
}
