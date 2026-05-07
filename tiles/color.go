package tiles

import (
	"image"
	"image/color"
)

var MonoModel = color.ModelFunc(monoConvert)

type Mono bool

func (m Mono) RGBA() (r, g, b, a uint32) {
	a = 0xFFFF
	if m {
		r, g, b = 0xFFFF, 0xFFFF, 0xFFFF
	}
	return
}

func (m Mono) Model() color.Model {
	return MonoModel
}

func (m Mono) At(x, y int) color.Color {
	return m
}

func (m Mono) Bounds() image.Rectangle {
	return image.Rect(-1e9, -1e9, 1e9, 1e9)
}

func monoConvert(c color.Color) color.Color {
	if _, ok := c.(Mono); ok {
		return c
	}

	r, g, b, _ := c.RGBA()
	if max(r, g, b) > 0xFFFF/2 {
		return Mono(true)
	}
	return Mono(false)
}

// BitAlpha is a color.Color with a single bit of alpha.
// That is, transparent or opaque.
type BitAlpha bool

func (ba BitAlpha) RGBA() (r, g, b, a uint32) {
	if ba {
		r, g, b = m, m, m
		a = m
	}
	return
}

var (
	BitAlphaModel = color.ModelFunc(bitAlphaModel)
)

func bitAlphaModel(c color.Color) color.Color {
	if b, ok := c.(BitAlpha); ok {
		return b
	}
	_, _, _, a := c.RGBA()
	if a > m/2 {
		return BitAlpha(true)
	}
	return BitAlpha(false)
}
