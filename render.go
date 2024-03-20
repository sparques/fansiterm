package fansiterm

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png"

	"golang.org/x/image/font/inconsolata"
	"golang.org/x/image/math/fixed"
)

// cursorRectFunc specifies a function for generating a rectanglular region to invert,
// for the purposes of rendering a cursor.
type cursorRectFunc func(image.Rectangle, int, int) image.Rectangle

// RenderRunes does not do *any* interpretation of escape codes or control characters like \r or \n.
// It simply renders a slice of runes (as a string) at the cursor position. It is up to the caller
// of RenderRunes to ensure there's enough space for the runes on the buffer and to process any
// control sequences.
func (d *Device) RenderRunes(sym []rune) {
	// TODO: replace r and c
	r, c := d.cursor.row, d.cursor.col
	if !d.attr.Reversed {
		d.Render.fontDraw.Src = d.attr.Fg
	} else {
		d.Render.fontDraw.Src = d.attr.Bg
	}

	// Which Face are we using? Note that useAltCharSet overrides Bold and Italics
	switch {
	case d.Render.useAltCharSet:
		d.Render.fontDraw.Face = d.Render.altCharSet
	case d.attr.Bold:
		d.Render.fontDraw.Face = inconsolata.Bold8x16
	default:
		d.Render.fontDraw.Face = inconsolata.Regular8x16
	}

	// draw background
	d.Clear(c, r, c+len(sym), r+1)

	// draw character
	// Ascent is pixels above baseline and descent is pixels below baseline. We want the bottom of the glyph aligned with bottom
	// of the cell
	d.Render.fontDraw.Dot = fixed.P(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r+1)-d.Render.fontDraw.Face.Metrics().Descent.Round())
	d.Render.fontDraw.DrawString(string(sym))

	// TODO: clean this mess up up; really could use better drawing routines
	// Need to do a performance comparison; would it be better to have "glyphs" that are lines and render those overtop
	// characters? The code would certainly be cleaner and simpler that way.
	if d.attr.Strike {
		// draw a single pixel high line through the center of the whole cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Render.cell.Max.Y/2,
				d.Render.cell.Max.X*len(sym),
				d.Render.cell.Max.Y/2+1).Add(image.Pt(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r))),
			d.Render.fontDraw.Src,
			image.Point{},
			draw.Src)
	}

	if d.attr.Underline || d.attr.DoubleUnderline {
		// draw a single pixel high line through the the whole cell, 3px above the bottom of the cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Render.cell.Max.Y-1,
				d.Render.cell.Max.X*len(sym),
				d.Render.cell.Max.Y).Add(image.Pt(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r))),
			d.Render.fontDraw.Src,
			image.Point{},
			draw.Src)
		// draw second line for double underline
		if d.attr.DoubleUnderline {
			draw.Draw(d.Render,
				image.Rect(
					0,
					d.Render.cell.Max.Y-3,
					d.Render.cell.Max.X*len(sym),
					d.Render.cell.Max.Y-2).Add(image.Pt(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r))),
				d.Render.fontDraw.Src,
				image.Point{},
				draw.Src)
		}
	}
}

func blockRect(cell image.Rectangle, c, r int) image.Rectangle {
	return cell.Add(image.Pt(cell.Max.X*c, cell.Max.Y*r))
}

func beamRect(cell image.Rectangle, c, r int) image.Rectangle {
	return image.Rect(0, 0, 1, cell.Max.Y).Add(image.Pt(cell.Max.X*c, cell.Max.Y*r))
}

func underscoreRect(cell image.Rectangle, c, r int) image.Rectangle {
	return image.Rect(0, cell.Max.Y-1, cell.Max.X, cell.Max.Y).Add(image.Pt(cell.Max.X*c, cell.Max.Y*r))
}

func (d *Device) cursorPt() image.Point {
	return image.Pt(d.Render.cell.Max.X*d.cursor.col, d.Render.cell.Max.Y*d.cursor.row)
}

func (d *Device) toggleCursor() {
	d.cursor.visible = !d.cursor.visible
	draw.Draw(d.Render,
		d.Render.cursorFunc(d.Render.cell, d.cursor.col, d.cursor.row),
		invertColors{d.Render},
		d.cursorPt(),
		draw.Src)
}

func (d *Device) hideCursor() {
	if d.cursor.visible {
		d.toggleCursor()
	}
}

func (d *Device) Image() image.Image {
	return d.Render // d.Render or d.Render.Image?
}

// Clear writes a block of current background color in a rectangular shape,
// specified in units of cells (rows and columns).
// So (*Device).Clear(0,0, (*Device).cols, (*Device).rows) would
// clear the whole screen.
func (d *Device) Clear(x1, y1, x2, y2 int) {
	if d.attr.Reversed {
		draw.Draw(d.Render,
			image.Rect(x1*d.Render.cell.Dx(), y1*d.Render.cell.Dy(), x2*d.Render.cell.Dx(), y2*d.Render.cell.Dy()),
			d.attr.Fg,
			image.Point{},
			draw.Src)
	} else {
		draw.Draw(d.Render,
			image.Rect(x1*d.Render.cell.Dx(), y1*d.Render.cell.Dy(), x2*d.Render.cell.Dx(), y2*d.Render.cell.Dy()),
			d.attr.Bg,
			image.Point{},
			draw.Src)
	}
}

// invertColors composites an image.Image, overriding the At() method
// so that colors returned are inverted. This is primarily for drawing
// the cursor.
type invertColors struct {
	image.Image
}

func (ic invertColors) At(x, y int) color.Color {
	r, g, b, a := ic.Image.At(x, y).RGBA()
	return color.RGBA{255 - uint8(r), 255 - uint8(g), 255 - uint8(b), uint8(a)}
}

// faintColors composites image.Image and draw.Image, overriding At() and Set()
// so that the alpha is half of the underlying image
type faintColors struct {
	image.Image
}

func (fc faintColors) At(x, y int) color.Color {
	r, g, b, _ := fc.Image.At(x, y).RGBA()
	return color.RGBA{uint8(r), uint8(g), uint8(b), uint8(0)}
}

/*
func (fc faintColors) Set(x, y int, c color.Color) {
	r, g, b, a := c.RGBA()
	fc.Image.Set(x, y, color.RGBA{uint8(r), uint8(g), uint8(b), uint8(a) / 2})
}
*/

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
