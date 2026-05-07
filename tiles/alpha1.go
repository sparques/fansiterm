package tiles

import (
	"image"
	"image/color"
)

// Alpha1 is a single bit-depth image.Image whose
// pixels are either opaque or transparent.
type Alpha1 struct {
	Pix    []uint8
	Stride int
	Rect   image.Rectangle
}

func NewAlpha1(r image.Rectangle) *Alpha1 {
	stride := bytesPerRow(r.Dx())
	return &Alpha1{
		Pix:    make([]uint8, stride*r.Dy()),
		Stride: stride,
		Rect:   r,
	}
}

func (a *Alpha1) ColorModel() color.Model {
	return BitAlphaModel
}

func (a *Alpha1) Bounds() image.Rectangle {
	return a.Rect
}

func (a *Alpha1) At(x, y int) color.Color {
	idx, bit := a.PixIdx(image.Pt(x, y))
	return BitAlpha(a.Pix[idx]&byte(bit) != 0)
}

func (a *Alpha1) Set(x, y int, c color.Color) {
	idx, bit := a.PixIdx(image.Pt(x, y))
	native := BitAlphaModel.Convert(c).(BitAlpha)
	if native {
		a.Pix[idx] |= byte(bit)
	} else {
		a.Pix[idx] &^= byte(bit)
	}
}

// PixIdx returns the index and bit-offset for the pixel at point p.
func (a *Alpha1) PixIdx(p image.Point) (idx int, offset int) {
	x := p.X - a.Rect.Min.X
	y := p.Y - a.Rect.Min.Y
	return y*a.Stride + x/8, 0x80 >> (x % 8)
}

func (a *Alpha1) Fill(rect image.Rectangle, c color.Color) {
	r := rect.Intersect(a.Rect)
	if r.Empty() {
		return
	}

	fillOn := bool(BitAlphaModel.Convert(c).(BitAlpha))

	minX := r.Min.X - a.Rect.Min.X
	maxX := r.Max.X - a.Rect.Min.X
	minY := r.Min.Y - a.Rect.Min.Y
	maxY := r.Max.Y - a.Rect.Min.Y

	startByte := minX >> 3
	endByte := (maxX - 1) >> 3

	startBit := minX & 7
	endBit := (maxX - 1) & 7

	startMask := byte(0xFF >> startBit)
	endMask := byte(0xFF << (7 - endBit))

	if startByte == endByte {
		mask := startMask & endMask
		for y := minY; y < maxY; y++ {
			i := y*a.Stride + startByte
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
	}

	for y := minY; y < maxY; y++ {
		row := y * a.Stride

		i0 := row + startByte
		if fillOn {
			a.Pix[i0] |= startMask
		} else {
			a.Pix[i0] &^= startMask
		}

		for i := i0 + 1; i < row+endByte; i++ {
			a.Pix[i] = full
		}

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
// Pixels scrolled off are discarded; newly revealed area is cleared.
func (a *Alpha1) Scroll(amount int) {
	h := a.Rect.Dy()
	if amount == 0 || h <= 0 {
		return
	}

	if amount >= h || amount <= -h {
		clear(a.Pix)
		return
	}

	stride := a.Stride

	if amount > 0 {
		src := amount * stride
		n := (h - amount) * stride
		copy(a.Pix[:n], a.Pix[src:src+n])
		clear(a.Pix[n : h*stride])
		return
	}

	amt := -amount
	dst := amt * stride
	n := (h - amt) * stride
	reverseCopy(a.Pix[dst:dst+n], a.Pix[:n])
	clear(a.Pix[:dst])
}

func bytesPerRow(width int) int {
	if width <= 0 {
		return 0
	}
	return (width + 7) / 8
}

func reverseCopy[E any](dst, src []E) {
	for i := min(len(dst), len(src)) - 1; i >= 0; i-- {
		dst[i] = src[i]
	}
}
