package fansiterm

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png"

	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

func (d *Device) RenderRune(sym rune) {
	r, c := d.offsetToRowCol(d.cursorPos)
	var bg image.Image
	if !d.attr.Reversed {
		d.fontDraw.Src = d.attr.Fg
		bg = d.attr.Bg
	} else {
		d.fontDraw.Src = d.attr.Bg
		bg = d.attr.Fg
	}

	// Which Face are we using? Note that useAltCharSet overrides Bold and Italics
	switch {
	case d.useAltCharSet:
		d.fontDraw.Face = d.altCharSet
	case d.attr.Bold:
		d.fontDraw.Face = inconsolata.Bold8x16
	default:
		d.fontDraw.Face = inconsolata.Regular8x16
	}

	// draw background
	draw.Draw(d.buf,
		d.cell.Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*(r))),
		bg,
		image.Point{},
		draw.Src)

	/* if d.useAltCharSet {
		draw.DrawMask(d.buf,
			d.cell.Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*(r))),
			d.fontDraw.Src,
			image.Point{},
			d.altCharSet[sym],
			image.Point{},
			draw.Over)
	} else { */

	// draw character
	// Ascent is pixels above baseline and descent is pixels below baseline. We want the bottom of the glyph aligned with bottom
	// of the cell
	//d.fontDraw.Dot = fixed.P(d.cell.Max.X*c, d.cell.Max.Y*(r+1)-d.fontDescent)
	d.fontDraw.Dot = fixed.P(d.cell.Max.X*c, d.cell.Max.Y*(r+1)-d.fontDraw.Face.Metrics().Descent.Round())
	d.fontDraw.DrawString(string([]rune{sym})) // sure hope this rune -> []rune -> string cast isn't as inefficient as it looks

	// TODO: clean this mess up up; really could use better drawing routines
	if d.attr.Strike {
		// draw a single pixel high line through the center of the whole cell
		draw.Draw(d.buf,
			image.Rect(0, d.cell.Max.Y/2, d.cell.Max.X, d.cell.Max.Y/2+1).Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*(r))),
			d.fontDraw.Src,
			image.Point{},
			draw.Src)
	}

	if d.attr.Underline {
		// draw a single pixel high line through the the whole cell, 3px above the bottom of the cell
		draw.Draw(d.buf,
			image.Rect(0, d.cell.Max.Y-1, d.cell.Max.X, d.cell.Max.Y).Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*(r))),
			d.fontDraw.Src,
			image.Point{},
			draw.Src)
	}

	if d.attr.DoubleUnderline {
		draw.Draw(d.buf,
			image.Rect(0, d.cell.Max.Y-3, d.cell.Max.X, d.cell.Max.Y-2).Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*(r))),
			d.fontDraw.Src,
			image.Point{},
			draw.Src)
		draw.Draw(d.buf,
			image.Rect(0, d.cell.Max.Y-1, d.cell.Max.X, d.cell.Max.Y).Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*(r))),
			d.fontDraw.Src,
			image.Point{},
			draw.Src)
	}

}

func (d *Device) toggleCursor() {
	d.cursorVisible = !d.cursorVisible
	r, c := d.offsetToRowCol(d.cursorPos)
	switch d.Config.CursorStyle {
	case CursorBlock:
		draw.Draw(d.buf,
			d.cell.Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*r)),
			invertColors{d.buf},
			image.Pt(d.cell.Max.X*c, d.cell.Max.Y*r),
			draw.Src)
	case CursorBeam:
		draw.Draw(d.buf,
			image.Rect(0, 0, 1, d.cell.Max.Y).Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*r)),
			invertColors{d.buf},
			image.Pt(d.cell.Max.X*c, d.cell.Max.Y*r),
			draw.Src)
	case CursorUnderscore:
		draw.Draw(d.buf,
			image.Rect(0, d.cell.Max.Y-1, d.cell.Max.X, d.cell.Max.Y).Add(image.Pt(d.cell.Max.X*c, d.cell.Max.Y*r)),
			invertColors{d.buf},
			image.Pt(d.cell.Max.X*c, d.cell.Max.Y*r+d.cell.Max.Y-1),
			draw.Src)
	}
}

func (d *Device) hideCursor() {
	if d.cursorVisible {
		d.toggleCursor()
	}
}

func (d *Device) Image() image.Image {
	return d.buf
}

// We could just compose draw.Image into Device...
// Why aren't we? (Because I'll have to update every reference to d.buf 😂😭)

// Set lets Device directly implement draw.Image
func (d *Device) Set(x, y int, c color.Color) {
	d.buf.Set(x, y, c)
}

func (d *Device) At(x, y int) color.Color {
	return d.buf.At(x, y)
}

func (d *Device) ColorModel() color.Model {
	return d.buf.ColorModel()
}

func (d *Device) Bounds() image.Rectangle {
	return d.buf.Bounds()
}

// Clear writes a block of current background color in a rectangular shape,
// specified in units of cells (rows and columns).
// So (*Device).Clear(0,0, (*Device).cols, (*Device).rows) would
// clear the whole screen.
func (d *Device) Clear(x1, y1, x2, y2 int) {
	draw.Draw(d.buf,
		image.Rect(x1*d.cell.Dx(), y1*d.cell.Dy(), x2*d.cell.Dx(), y2*d.cell.Dy()),
		image.NewUniform(d.attr.Bg),
		image.Point{},
		draw.Src)
}

type invertColors struct {
	image.Image
}

func (ic invertColors) At(x, y int) color.Color {
	r, g, b, a := ic.Image.At(x, y).RGBA()
	return color.RGBA{255 - uint8(r), 255 - uint8(g), 255 - uint8(b), uint8(a)}
}

// imageTranslate works a bit like the Subimage() method on various image package
// objects. However, it wraps a draw.Image allowing both calls to Set() and At().
type imageTranslate struct {
	draw.Image
	offset image.Point
}

// NewImageTranslate
func NewImageTranslate(offset image.Point, img draw.Image) *imageTranslate {
	return &imageTranslate{
		offset: offset,
		Image:  img,
	}
}

func (it *imageTranslate) Set(x, y int, c color.Color) {
	it.Image.Set(x+it.offset.X, y+it.offset.Y, c)
}

func (it *imageTranslate) At(x, y int) color.Color {
	return it.Image.At(x+it.offset.X, y+it.offset.Y)
}

func (it imageTranslate) Bounds() image.Rectangle {
	return it.Image.Bounds().Sub(it.offset)
}

// Color both implements color.Color and image.Image
// It's like image.Uniform.
type Color struct {
	rgba color.RGBA
}

func NewOpaqueColor(r, g, b uint8) Color {
	return Color{color.RGBA{r, g, b, 255}}
}

func NewColor(r, g, b, a uint8) Color {
	return Color{color.RGBA{r, g, b, a}}
}

func (c Color) RGBA() (r, g, b, a uint32) {
	return c.rgba.RGBA()
}

func (c Color) At(int, int) color.Color {
	return c.rgba
}

func (c Color) Bounds() image.Rectangle {
	return image.Rectangle{image.Point{-1e9, -1e9}, image.Point{1e9, 1e9}}
}

func (c Color) ColorModel() color.Model {
	return c
}

func (c Color) Convert(c2 color.Color) color.Color {
	return c2
}
