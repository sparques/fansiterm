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

func (d *Device) UpdateAttr() {
	d.updateAttr()
}

// updateAttr updates d.Render.active based on d.Attr.
// TODO a name that doesn't suck
func (d *Device) updateAttr() {
	d.Render.active.fg, d.Render.active.bg = d.attr.Fg, d.attr.Bg
	if d.attr.Reversed {
		d.Render.active.fg, d.Render.active.bg = d.attr.Bg, d.attr.Fg
	}

	// Bold and Itallic override G0 and G1
	switch {
	case d.attr.Bold:
		d.Render.active.tileSet = d.Render.BoldCharSet
	case d.attr.Italic:
		d.Render.active.tileSet = d.Render.ItalicCharSet
	case d.attr.ShiftOut:
		d.Render.active.tileSet = d.Render.G1
	default:
		d.Render.active.tileSet = d.Render.G0
	}
}

// cursorPt returns the location of the cursor as an image.Point
// From perspective of viewing the rendered terminal, this is the top left corner of the cell the cursor is in.
func (d *Device) cursorPt() image.Point {
	return image.Pt(d.Render.Bounds().Min.X+d.Render.cell.Dx()*d.cursor.col, d.Render.Bounds().Min.Y+d.Render.cell.Dy()*d.cursor.row)
}

// RenderRune does not do *any* interpretation of escape codes or control characters like \r or \n.
// It simply renders a single rune at the cursor position. It is up to the caller
// of RenderRune to process any control sequences / handle non-printing characters.
func (d *Device) RenderRune(sym rune) {
	d.Render.active.tileSet.DrawTile(sym, d.Render.Image, d.cursorPt(), d.Render.active.fg, d.Render.active.bg)

	if d.attr.Strike {
		// draw a single pixel high line through the center of the whole cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Render.cell.Dy()/2+1,
				d.Render.cell.Dx(),
				d.Render.cell.Dy()/2+2).Add(d.cursorPt()),
			d.Render.active.fg,
			image.Point{},
			draw.Src)
	}

	if d.attr.Underline {
		// draw a single pixel high line through the the whole cell, 3px above the bottom of the cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Render.cell.Dy()-1,
				d.Render.cell.Dx(),
				d.Render.cell.Dy()).Add(d.cursorPt()),
			d.Render.active.fg,
			image.Point{},
			draw.Src)
		// draw second line for double underline
		if d.attr.DoubleUnderline {
			draw.Draw(d.Render,
				image.Rect(
					0,
					d.Render.cell.Dy()-3,
					d.Render.cell.Dx(),
					d.Render.cell.Dy()-2).Add(d.cursorPt()),
				d.Render.active.fg,
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

func (d *Device) clearAll() {
	// if underlying Image supports Fill(), use that instead
	if fillable, ok := d.Render.Image.(gfx.Filler); ok {
		fillable.Fill(d.Render.Bounds(), d.attr.Bg)
		return
	}

	draw.Draw(d.Render,
		d.Render.Bounds(),
		d.attr.Bg,
		image.Point{},
		draw.Src)
}
