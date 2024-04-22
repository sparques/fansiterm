package fansiterm

import (
	"image"
	"image/draw"
	_ "image/png"

	"github.com/sparques/fansiterm/xform"
	"github.com/sparques/gfx"
)

// cursorRectFunc specifies a function for generating a rectanglular region to invert,
// for the purposes of rendering a cursor.
type cursorRectFunc func(image.Rectangle, image.Point) image.Rectangle

var (
	// CursorBlock, CursorBeam, and CursorUnderscore are the 3 cursor display options.
	CursorBlock      = blockRect
	CursorBeam       = beamRect
	CursorUnderscore = underscoreRect
)

// cursorPt returns the location of the cursor as an image.Point
// From perspective of viewing the rendered terminal, this is the top left corner of the cell the cursor is in.
func (d *Device) cursorPt() image.Point {
	return image.Pt(d.Render.Bounds().Min.X+d.Render.cell.Dx()*d.cursor.col, d.Render.Bounds().Min.Y+d.Render.cell.Dy()*d.cursor.row)
}

// RenderRunes does not do *any* interpretation of escape codes or control characters like \r or \n.
// It simply renders a slice of runes (as a string) at the cursor position. It is up to the caller
// of RenderRunes to ensure there's enough space for the runes on the buffer and to process any
// control sequences.
func (d *Device) RenderRunes(sym []rune) {
	fg, bg, ts := d.attr.Fg, d.attr.Bg, d.Render.charSet
	// Which Face are we using? Note that useAltCharSet overrides Bold and Italics
	switch {
	case d.Render.useAltCharSet:
		ts = d.Render.altCharSet
	case d.attr.Bold:
		ts = d.Render.boldCharSet
	case d.attr.Italic:
		ts = d.Render.italicCharSet
	}

	if d.attr.Reversed {
		fg, bg = bg, fg
	}

	// consider making this work; then you can do bold + Italic
	// if d.attr.Italic {
	// 	ts = Italics{ts}
	// }

	// draw characters
	for i, glyph := range sym {
		ts.DrawTile(glyph, d.Render.Image,
			d.cursorPt().Add(image.Pt(i*d.Render.cell.Dx(), 0)),
			fg, bg)
	}

	// TODO: clean this mess up up; really could use better drawing routines
	// Need to do a performance comparison; would it be better to have "glyphs" that are lines and render those overtop
	// characters? The code would certainly be cleaner and simpler that way.
	if d.attr.Strike {
		// draw a single pixel high line through the center of the whole cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Render.cell.Max.Y/2+1,
				d.Render.cell.Max.X*len(sym),
				d.Render.cell.Max.Y/2+2).Add(d.cursorPt()),
			fg,
			image.Point{},
			draw.Src)
	}

	if d.attr.Underline {
		// draw a single pixel high line through the the whole cell, 3px above the bottom of the cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Render.cell.Max.Y-1,
				d.Render.cell.Max.X*len(sym),
				d.Render.cell.Max.Y).Add(d.cursorPt()),
			fg,
			image.Point{},
			draw.Src)
		// draw second line for double underline
		if d.attr.DoubleUnderline {
			draw.Draw(d.Render,
				image.Rect(
					0,
					d.Render.cell.Max.Y-3,
					d.Render.cell.Max.X*len(sym),
					d.Render.cell.Max.Y-2).Add(d.cursorPt()),
				fg,
				image.Point{},
				draw.Src)
		}
	}
}

func blockRect(cell image.Rectangle, pt image.Point) image.Rectangle {
	return cell.Add(pt)
}

func beamRect(cell image.Rectangle, pt image.Point) image.Rectangle {
	return image.Rect(0, 0, 1, cell.Max.Y).Add(pt)
}

func underscoreRect(cell image.Rectangle, pt image.Point) image.Rectangle {
	return image.Rect(0, cell.Max.Y-1, cell.Max.X, cell.Max.Y).Add(pt)
}

func (d *Device) toggleCursor() {
	rect := d.Render.cursorFunc(d.Render.cell, d.cursorPt())
	d.cursor.visible = !d.cursor.visible

	draw.Draw(d.Render,
		rect,
		xform.InvertColors(d.Render),
		rect.Min, // must align rect in src to same position
		draw.Src)
}

func (d *Device) hideCursor() {
	if d.cursor.visible {
		d.toggleCursor()
	}
}

func (d *Device) showCursor() {
	if d.cursor.show && !d.cursor.visible {
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
	rect := image.Rect(
		x1*d.Render.cell.Dx(), y1*d.Render.cell.Dy(),
		x2*d.Render.cell.Dx(), y2*d.Render.cell.Dy()).
		Add(d.Render.Bounds().Min)

	// if underlying Image supports Fill(), use that instead
	if fillable, ok := d.Render.Image.(gfx.Filler); ok {
		fillable.Fill(rect, d.attr.Bg)
		return
	}

	draw.Draw(d.Render,
		rect,
		d.attr.Bg,
		image.Point{},
		draw.Src)

}
