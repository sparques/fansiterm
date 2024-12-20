package fansiterm

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png"

	"github.com/mattn/go-runewidth"
	"github.com/sparques/fansiterm/tiles"
	"github.com/sparques/fansiterm/xform"
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

var unicode = runewidth.NewCondition()

type Render struct {
	draw.Image
	colorSystem *colorSystem
	bounds      image.Rectangle
	active      struct {
		tileSet *tiles.Tiler
		fg      Color
		bg      Color
		// G tracks our character sets, for now 0 and 1
		g []*tiles.Tiler
		// tracking shift-in/out
		shift int
	}
	CharSet       tiles.Tiler
	AltCharSet    tiles.Tiler
	BoldCharSet   tiles.Tiler
	ItalicCharSet tiles.Tiler
	cell          image.Rectangle
	cursorFunc    cursorRectFunc
	// DisplayFunc is called after a write to the terminal. This is for some displays that require a flush / blit / sync call.
	DisplayFunc func()

	scroll       func(int)
	regionScroll func(image.Rectangle, int)
	vectorScroll func(image.Rectangle, image.Point)
	fill         func(image.Rectangle, color.Color)
}

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
func (d *Device) RenderRune(sym rune) (width int) {
	width = 1
	if sym > 255 {
		// do runewidth check and adjusted width as necessary
		width = unicode.RuneWidth(sym)
	}

	if width == 0 {
		// FIXME: corner case of using a zero-width (combining) character
		// when we're in the last column
		(*d.Render.active.tileSet).DrawTile(sym, d.Render.Image, d.cursorPt().Add(image.Pt(-d.Render.cell.Dx(), 0)), d.Render.active.fg, color.Alpha{0})
	} else {
		(*d.Render.active.tileSet).DrawTile(sym, d.Render.Image, d.cursorPt(), d.Render.active.fg, d.Render.active.bg)
	}

	if d.attr.Strike {
		// draw a single pixel high line through the center of the whole cell
		draw.Draw(d.Render,
			image.Rect(
				0,
				d.Config.StrikethroughHeight,
				d.Render.cell.Dx()*width,
				d.Config.StrikethroughHeight+1,
			).Add(d.cursorPt()),
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
				d.Render.cell.Dx()*width,
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
					d.Render.cell.Dx()*width,
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

	return
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

// func (d *Device) Fill(region image.Rectangle, c color.Color) {
// 	region = region.Add(d.Render.bounds.Min).Intersect(d.Render.bounds)
//
// 	if fillable, ok := d.Render.Image.(gfx.Filler); ok {
// 		fillable.Fill(region, c)
// 		return
// 	}
// 	draw.Draw(d.Render, region, image.NewUniform(c), region.Min, draw.Src)
// }

// Clear writes a block of current background color in a rectangular shape,
// specified in units of cells (rows and columns).
// So (*Device).Clear(0,0, (*Device).cols, (*Device).rows) would
// clear the whole screen.
func (d *Device) Clear(x1, y1, x2, y2 int) {
	rect := image.Rect(
		x1*d.Render.cell.Dx(), y1*d.Render.cell.Dy(),
		x2*d.Render.cell.Dx(), y2*d.Render.cell.Dy())

	d.Render.Fill(rect, d.attr.Bg)
}

func (d *Device) clearAll() {
	d.Render.Fill(d.Render.bounds, d.attr.Bg)
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

// should probably move to gfx package
func softRegionScroll(img draw.Image, region image.Rectangle, amount int) {
	softVectorScroll(img, region, image.Pt(0, amount))
}

func softVectorScroll(img draw.Image, region image.Rectangle, vector image.Point) {
	region = img.Bounds().Intersect(region)
	var dst, src image.Point

	for y := range region.Dy() {
		if vector.Y >= 0 {
			dst.Y = region.Min.Y + y
		} else {
			dst.Y = region.Max.Y - (y + 1)
		}
		for x := range region.Dx() {
			if vector.X >= 0 {
				dst.X = region.Min.X + x
			} else {
				dst.X = region.Max.X - (x + 1)
			}
			src = dst.Add(vector).Mod(region)
			img.Set(dst.X, dst.Y, img.At(src.X, src.Y))
		}
	}

	return
}

func (r *Render) Scroll(pixAmt int) {
	r.scroll(pixAmt)
}

func (r *Render) RegionScroll(region image.Rectangle, pixAmt int) {
	r.regionScroll(region, pixAmt)
}

func (r *Render) VectorScroll(region image.Rectangle, vector image.Point) {
	r.vectorScroll(region, vector)
}

func (r *Render) Fill(region image.Rectangle, c color.Color) {
	region = region.Add(r.bounds.Min).Intersect(r.bounds)
	r.fill(region, c)
}
