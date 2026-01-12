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

// Alpha1 is a single bit-depth image.Image whose
// pixels are either opaque or transparent.
type Alpha1 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewAlpha1(r image.Rectangle) *Alpha1 {
	return &Alpha1{
		Pix:    make([]uint8, r.Dx()*r.Dy()/8),
		Stride: r.Dx(),
		Rect:   r,
	}
}

func (a *Alpha1) ColorModel() color.Model {
	return BitAlphaModel
}

func (a *Alpha1) Bounds() image.Rectangle {
	return a.Rect
}

func (a *Alpha1) At(x, y int) (c color.Color) {
	return BitAlpha((a.Pix[y*a.Stride+x/8]<<(x%8))&0x80 == 0x80)
}

func (a *Alpha1) Set(x, y int, c color.Color) {
	native := BitAlphaModel.Convert(c).(BitAlpha)

	if native {
		a.Pix[y*a.Stride+x/8] |= 0x80 >> (x % 8)
	} else {
		a.Pix[y*a.Stride+x/8] &= ^(0x80 >> (x % 8))
	}
}

// PixIdx returns the index and bit-offset for the pixel at
// point p.
func (a *Alpha1) PixIdx(p image.Point) (idx int, offset int) {
	return p.Y*a.Stride + p.X/8, 0x80 >> (p.X % 8)
}

func (a *Alpha1) Fill(rect image.Rectangle, c color.Color) {
	r := rect.Intersect(a.Rect)
	if r.Empty() {
		return
	}

	fillOn := bool(BitAlphaModel.Convert(c).(BitAlpha))

	minX, minY := r.Min.X, r.Min.Y
	maxX, maxY := r.Max.X, r.Max.Y

	startByte := minX >> 3
	endByte := (maxX - 1) >> 3 // inclusive

	startBit := minX & 7
	endBit := (maxX - 1) & 7

	startMask := byte(0xFF >> startBit)
	endMask := byte(0xFF << (7 - endBit))

	if startByte == endByte {
		mask := startMask & endMask
		for y := minY; y < maxY; y++ {
			i := (y-a.Rect.Min.Y)*a.Stride + startByte
			if fillOn {
				a.Pix[i] |= mask
			} else {
				a.Pix[i] &^= mask
			}
		}
		return
	}

	var full byte
	if fillOn {
		full = 0xFF
	} else {
		full = 0x00
	}

	for y := minY; y < maxY; y++ {
		row := (y - a.Rect.Min.Y) * a.Stride

		// first partial byte
		i0 := row + startByte
		if fillOn {
			a.Pix[i0] |= startMask
		} else {
			a.Pix[i0] &^= startMask
		}

		// middle full bytes
		for i := i0 + 1; i < row+endByte; i++ {
			a.Pix[i] = full
		}

		// last partial byte
		i1 := row + endByte
		if fillOn {
			a.Pix[i1] |= endMask
		} else {
			a.Pix[i1] &^= endMask
		}
	}
}

// Scroll scrolls the image by amount pixels vertically.
// Positive amount scrolls the "screen" down / moves the image up.
// Pixels scrolled off are discarded; newly revealed area is cleared (inactive).
func (a *Alpha1) Scroll(amount int) {
	h := a.Rect.Dy()
	if amount == 0 || h <= 0 {
		return
	}

	// Clamp huge scrolls: just clear everything.
	if amount >= h || amount <= -h {
		for i := range a.Pix {
			a.Pix[i] = 0
		}
		return
	}

	stride := a.Stride

	if amount > 0 {
		// Move image up by 'amount': dst rows [0 .. h-amount) come from src [amount .. h)
		src := amount * stride
		n := (h - amount) * stride
		copy(a.Pix[0:n], a.Pix[src:src+n])

		// Clear newly revealed bottom area: rows [h-amount .. h)
		clearStart := n
		clearEnd := h * stride
		for i := clearStart; i < clearEnd; i++ {
			a.Pix[i] = 0
		}
		return
	}

	// amount < 0: move image down by 'amt'
	amt := -amount
	dst := amt * stride
	n := (h - amt) * stride
	reverseCopy(a.Pix[dst:dst+n], a.Pix[0:n])

	// Clear newly revealed top area: rows [0 .. amt)
	clearEnd := dst
	for i := 0; i < clearEnd; i++ {
		a.Pix[i] = 0
	}
}

func reverseCopy[E any](dst, src []E) {
	for i := min(len(dst), len(src)) - 1; i >= 0; i-- {
		dst[i] = src[i]
	}
}
