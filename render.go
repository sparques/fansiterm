package fansiterm

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png"

	"github.com/sparques/fansiterm/tiles"
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

	switch d.Render.active.g[d.Render.active.shift] {
	case &d.Render.CharSet:
		switch {
		case d.attr.Bold:
			d.Render.active.tileSet = &d.Render.BoldCharSet
		case d.attr.Italic:
			d.Render.active.tileSet = &d.Render.ItalicCharSet
		default:
			d.Render.active.tileSet = d.Render.active.g[d.Render.active.shift]
		}
	case &d.Render.AltCharSet:
		// altCharSet in use
		var ts tiles.Tiler
		switch {
		case d.attr.Bold:
			ts = tiles.NewMultiTileSet(d.Render.AltCharSet, d.Render.BoldCharSet)
		case d.attr.Italic:
			ts = tiles.NewMultiTileSet(d.Render.AltCharSet, d.Render.ItalicCharSet)
		default:
			ts = d.Render.AltCharSet
		}
		d.Render.active.tileSet = &ts
	default:
	}

	return
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
	(*d.Render.active.tileSet).DrawTile(sym, d.Render.Image, d.cursorPt(), d.Render.active.fg, d.Render.active.bg)

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

	// conceal should be last
	if d.attr.Conceal {
		draw.Draw(d.Render,
			d.Render.cell.Bounds().Add(d.cursorPt()),
			xform.Blur(d.Render),
			d.cursorPt(),
			draw.Src)
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

func (d *Device) Fill(region image.Rectangle, c color.Color) {
	region = region.Add(d.Render.bounds.Min).Intersect(d.Render.bounds)

	if fillable, ok := d.Render.Image.(gfx.Filler); ok {
		fillable.Fill(region, c)
		return
	}
	draw.Draw(d.Render, region, image.NewUniform(c), region.Min, draw.Src)
}

// Clear writes a block of current background color in a rectangular shape,
// specified in units of cells (rows and columns).
// So (*Device).Clear(0,0, (*Device).cols, (*Device).rows) would
// clear the whole screen.
func (d *Device) Clear(x1, y1, x2, y2 int) {
	rect := image.Rect(
		x1*d.Render.cell.Dx(), y1*d.Render.cell.Dy(),
		x2*d.Render.cell.Dx(), y2*d.Render.cell.Dy())

	d.Fill(rect, d.attr.Bg)
}

func (d *Device) clearAll() {
	d.Fill(d.Render.bounds, d.attr.Bg)
}

// Bounds returns the image.Rectangle that aligns with terminal cell boundaries
func (r Render) Bounds() image.Rectangle {
	return r.bounds
}

func (r Render) Set(x, y int, c color.Color) {
	if !image.Pt(x, y).In(r.bounds) {
		return
	}
	r.Image.Set(x, y, c)
}
