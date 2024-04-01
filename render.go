package fansiterm

import (
	"image"
	"image/draw"
	_ "image/png"
)

// cursorRectFunc specifies a function for generating a rectanglular region to invert,
// for the purposes of rendering a cursor.
type cursorRectFunc func(image.Rectangle, int, int) image.Rectangle

// cursorPt returns the location of the cursor as an image.Point
// From perspective of viewing the rendered terminal, this is the top left corner.
func (d *Device) cursorPt() image.Point {
	return image.Pt(d.Render.cell.Max.X*d.cursor.col, d.Render.cell.Max.Y*d.cursor.row)
}

// RenderRunes does not do *any* interpretation of escape codes or control characters like \r or \n.
// It simply renders a slice of runes (as a string) at the cursor position. It is up to the caller
// of RenderRunes to ensure there's enough space for the runes on the buffer and to process any
// control sequences.
func (d *Device) RenderRunes(sym []rune) {
	// TODO: replace r and c
	r, c := d.cursor.row, d.cursor.col

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
				d.Render.cell.Max.Y/2,
				d.Render.cell.Max.X*len(sym),
				d.Render.cell.Max.Y/2+1).Add(image.Pt(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r))),
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
				d.Render.cell.Max.Y).Add(image.Pt(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r))),
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
					d.Render.cell.Max.Y-2).Add(image.Pt(d.Render.cell.Max.X*c, d.Render.cell.Max.Y*(r))),
				fg,
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

func (d *Device) toggleCursor() {
	d.cursor.visible = !d.cursor.visible
	rect := d.Render.cursorFunc(d.Render.cell, d.cursor.col, d.cursor.row)
	draw.Draw(d.Render,
		rect,
		invertColors{d.Render},
		rect.Min, // must align rect in src to same position
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
