package fansiterm

import (
	"bytes"
	"image"
	"image/draw"
	"io"
	"sync"
	"time"

	"github.com/sparques/fansiterm/tiles"
	"github.com/sparques/fansiterm/tiles/drawing"
	"github.com/sparques/fansiterm/tiles/sweet16"
	"github.com/sparques/fansiterm/xform"
	"github.com/sparques/gfx"
)

// Device implements a virtual terminal. It supports being io.Write()n to. It handles the cursor and processing of
// sequences.
//
//go:export
type Device struct {
	// BellFunc is called if it is non-null and the terminal would
	// display a bell character
	// TODO: Implement affirmative beep (default) and negative acknowledge beep
	// Negative acknowledge is produced when \a is sent while in SHIFT-OUT mode.
	// Affirmative: C-G (quarter notes?)
	// NAK: C♭ (whole note?)
	BellFunc func()

	// Config species the runtime configurable features of fansiterm.
	Config Config

	// cols and rows specify the size in characters of the terminal.
	cols, rows int

	// cursor  collects together all the fields for handling the cursor.
	cursor Cursor

	// attr tracks the currently applied attributes
	attr Attr

	// attrDefault is used when attr is zero-value or nil
	attrDefault Attr

	// scrollRegion defines what part of the screen should scroll
	// by default it is an empty image.Rectangle which means scroll
	// the whole screen.
	scrollArea   image.Rectangle
	scrollRegion [2]int

	// Render collects together all the graphical rendering fields.
	Render Render

	// inputBuf buffers chracters between write calls. This is exclusively used to
	// buffer incomplete escape sequences.
	inputBuf []rune

	// Miscellaneous properties, like "Window Title"
	Properties map[Property]string

	// Output specifies the program attached to the terminal. This should be the
	// same interface that the input mechanism (whatever that may be) uses to write
	// to the program. On POSIX systems, this would be equivalent to Stdin.
	// Default is io.Discard. Setting to nil will cause Escape Sequences that
	// write a response to panic.
	Output io.Writer

	sync.Mutex
}

// Cursor is used to track the cursor.
type Cursor struct {
	// col is the current column. This is zero indexed.
	col int
	// row is the current row. This is zero indexed.
	row int
	// show is whether we should be showing the the cursor.
	show bool
	// visible is whether or not the cursor is currently visible. When rendering text,
	// we hide the cursor, then re-enable it when done.
	visible bool

	// prevPos is for saving cursor position; The indicies are col, row.
	prevPos [2]int
}

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
	// DisplayFunc is called after a write to the terminal. This is for some displays require a flush / blit / sync call.
	DisplayFunc func()
}

type Config struct {
	TabSize     int
	CursorStyle int
	CursorBlink bool
	BoldColors  bool
}

type Attr struct {
	Bold            bool
	Underline       bool
	DoubleUnderline bool
	Strike          bool
	Blink           bool
	Reversed        bool
	Italic          bool
	Conceal         bool
	Fg              Color
	Bg              Color
}

var ConfigDefault = Config{
	TabSize:    8,
	BoldColors: true,
}

var altToUnicode = map[rune]rune{
	'0':  9608, // block
	0x61: 9618, // 50% block
	0x68: 9617, // 25% block
	0x6a: 9496, // bottom right corner
	0x6b: 9488, // top right corner
	0x6c: 9484, // top left corner
	0x6d: 9492, // bottom left corner
	0x6e: 9532, // cross
	'q':  9472, // horizontal
	'r':  9472, // horizontal

	0x74: 9500,   // T right
	0x75: 9508,   // T left
	0x76: 9524,   // T up
	0x77: 9516,   // T down
	0x78: 0x2502, // vertical
}

// New returns an initialized *Device. If buf is nil, an internal buffer is used. Otherwise
// if you specify a hardware backed draw.Image, writes to Device will immediately be written
// to the backing hardware--whether this is instaneous or buffered is up to the device and the
// device driver.
func New(cols, rows int, buf draw.Image) *Device {
	// Eventually I'd like to support different fonts and dynamic resizing
	// I'm trying to get to an MVP first.
	// thus, hardcoded font face
	// 7x13 is smaller and non-antialiased. For small screens it might be a better choice
	// than the 8x13 pre-render of inconsolata, however it doesn't have as many unicode-glyps
	// as inconsolata.
	//fontFace := basicfont.Face7x13
	cell := image.Rect(0, 0, 8, 16)

	if buf == nil {
		buf = image.NewRGBA(image.Rect(0, 0, cols*cell.Max.X, rows*cell.Max.Y))
	}

	// yoink the color model to init our colorSystem
	colorSystem := NewColorSystem(buf.ColorModel())

	// figure out our actual terminal bounds.
	bounds := image.Rect(0, 0, cell.Dx()*cols, cell.Dy()*rows).Add(buf.Bounds().Min)

	// if our backing buffer is bigger than our grid of cells, center the terminal
	// ... more or less.

	// figure out how much we need to shift around
	offset := image.Pt((buf.Bounds().Dx()%cell.Dx())/2, (buf.Bounds().Dy()%cell.Dy())/2)

	// shift around
	bounds = bounds.Add(offset)

	charSet := tiles.NewMultiTileSet(sweet16.Regular8x16, drawing.TileSet)
	altCharSet := tiles.NewRemap(charSet)
	altCharSet.Map = altToUnicode

	d := &Device{
		cols: cols,
		rows: rows,
		Render: Render{
			Image:         buf,
			colorSystem:   colorSystem,
			bounds:        bounds,
			AltCharSet:    altCharSet,
			CharSet:       charSet,
			BoldCharSet:   sweet16.Bold8x16,
			ItalicCharSet: &tiles.Italics{FontTileSet: sweet16.Bold8x16},
			cell:          cell,
			cursorFunc:    blockRect,
		},
		cursor: Cursor{
			show: true,
		},
		Config:       ConfigDefault,
		Output:       io.Discard,
		Properties:   make(map[Property]string),
		scrollRegion: [2]int{0, rows - 1},
	}

	// Establish defaults
	d.attrDefault.Fg = colorSystem.PaletteANSI[7]
	d.attrDefault.Bg = colorSystem.PaletteANSI[0]

	// only pre-fill our area. If user wants the rest of the buffer colored in, that's
	// on them.
	d.Fill(bounds, d.attrDefault.Bg)

	d.Render.active.g = make([]*tiles.Tiler, 2)
	d.Reset()
	d.updateAttr()

	return d
}

// NewAtResolution is like New, but rather than specifying the columns and rows,
// you specify the desired resolution. The maximum rows and cols will be determined
// automatically and the terminal rendered in the center.
// Fansiterm will only ever update / work on the rectangle it has claimed.
// If you want to use an existing backing buffer and position that, use NewWithBuf and
// use xform.SubImage() to locate the terminal.

func NewAtResolution(x, y int, buf draw.Image) *Device {
	// TODO: This is a crappy way of figuring out what font we're using. Do something else.
	d := New(1, 1, nil)
	// use d.Render.cell to figure out rows and cols; integer division will round down
	// which is what we want
	cols := x / d.Render.cell.Max.X
	rows := y / d.Render.cell.Max.Y

	if buf == nil {
		buf = image.NewRGBA(image.Rect(0, 0, x, y))
	}

	return New(cols, rows, buf)
}

// NewWithBuf uses buf as its target. NewWithBuf() will panic if called against a
// nil buf. If using fansiterm with backing hardware, NewWithBuf is likely the way
// you want to instantiate fansiterm.
// If you have buf providing an interface to a 240x135 screen, using the default
// 8x16 tiles, you can have an 40x8 cell terminal, with 7 rows of pixels leftover.
// If you want to have those extra 7 rows above the rendered terminal, you can do
// so like this:
//
// term := NewWithBuf(xform.SubImage(buf,image.Rect(0,0,240,128).Add(0,7)))
//
// Note: you can skip the Add() and just define your rectangle as
// image.Rect(0,7,240,135), but I find supplying the actual dimensions and then
// adding an offset to be clearer.
func NewWithBuf(buf draw.Image) *Device {
	if buf == nil {
		panic("NewWithBuf must be called with non-nil buf")
	}

	// TODO: How do I dynamically do this in a way that makes sense?
	cols := buf.Bounds().Dx() / 8
	rows := buf.Bounds().Dy() / 16

	return New(cols, rows, buf)
}

// TODO: FIXME
func (d *Device) HandleResize() {
	cols := d.Render.Bounds().Dx() / d.Render.cell.Dx()
	rows := d.Render.Bounds().Dy() / d.Render.cell.Dy()

	//draw.Draw(buf, buf.Bounds(), image.Black, image.Point{}, draw.Src)

	d.cols = cols
	d.rows = rows

	offset := image.Pt((d.Render.Bounds().Dx()%d.Render.cell.Dx())/2, (d.Render.Bounds().Dy()%d.Render.cell.Dy())/2)

	if !(offset.X == 0 && offset.Y == 0) {
		// first save a copy
		orig := image.NewRGBA(d.Render.Bounds())
		draw.Draw(orig, orig.Bounds(), d.Render.Image, orig.Bounds().Min, draw.Src)

		// clear whole thing
		draw.Draw(d.Render, d.Render.Bounds(), d.attr.Bg, d.Render.Bounds().Min, draw.Src)

		// Setup the offset
		d.Render.Image = xform.SubImage(d.Render.Image, image.Rect(0, 0, cols*d.Render.cell.Dx(), rows*d.Render.cell.Dy()).Add(offset))

		// write copy to offset-image
		draw.Draw(d.Render, d.Render.Bounds(), orig, orig.Bounds().Min, draw.Src)
	}
}

func (d *Device) Reset() {
	d.attr = d.attrDefault
	d.Render.active.g[0] = &d.Render.CharSet
	d.Render.active.g[1] = &d.Render.AltCharSet
	d.Render.active.tileSet = d.Render.active.g[0]
	d.clearAll()
	d.MoveCursorAbs(1, 1)
	d.scrollArea = image.Rectangle{}
	d.scrollRegion = [2]int{0, d.rows - 1}
}

// SetCursorStyle changes the shape of the cursor. Valid options are CursorBlock,
// CursorBeam, and CursorUnderscore. CursorBlock is the default.
func (d *Device) SetCursorStyle(style cursorRectFunc) {
	d.hideCursor()
	d.Render.cursorFunc = style
	d.showCursor()
}

func (d *Device) SetAttrDefault(attr Attr) {
	d.attrDefault = attr
}

// VisualBell inverts the screen for a tenth of a second.
func (d *Device) VisualBell() {
	draw.Draw(d.Render.Image, d.Render.Bounds(), xform.InvertColors(d.Render.Image), d.Render.Bounds().Min, draw.Src)
	time.Sleep(time.Second / 10)
	draw.Draw(d.Render.Image, d.Render.Bounds(), xform.InvertColors(d.Render.Image), d.Render.Bounds().Min, draw.Src)
}

// WriteAt works like calling the save cursor position escape sequence, then
// the absolute set cursor position escape sequence, writing to the terminal,
// and then finally restoring cursor position. The offset is just the i'th
// character on screen. Negative offset values are set to 0, values larger than
// d.rows * d.cols are set to d.rows*d.cols-1.
func (d *Device) WriteAt(p []byte, off int64) (n int, err error) {
	col, row := d.cursor.col, d.cursor.row
	defer func() {
		d.hideCursor()
		d.cursor.col = col
		d.cursor.row = row
		d.showCursor()
	}()
	if d.cursor.visible {
		d.toggleCursor()
	}
	off = bound(off, 0, int64(d.rows*d.cols)-1)
	d.cursor.row = int(off) / d.cols
	d.cursor.col = int(off) % d.cols
	return d.Write(p)
}

func isControl(r rune) bool {
	return r < 0x20
}

func isFinal(r rune) bool {
	return r >= 0x40
}

// Write implements io.Write and is the main way to interract with a (*fansiterm).Device. This is
// essentially writing to the "terminal."
// Writes are more or less unbuffered with the exception of escape sequences. If a partial escape sequence
// is written to Device, the beginning will be bufferred and prepended to the next write.
// Certain broken escape sequence can potentially block forever.
func (d *Device) Write(data []byte) (n int, err error) {
	d.Lock()
	defer d.Unlock()

	if d.Render.DisplayFunc != nil {
		defer d.Render.DisplayFunc()
	}

	runes := bytes.Runes(data)

	// first un-invert cursor (if we're showing it)
	if d.cursor.visible {
		d.toggleCursor()
	}
	defer d.showCursor()

	if len(d.inputBuf) != 0 {
		runes = append(d.inputBuf, runes...)
		d.inputBuf = []rune{}
	}

	// var endIdx int
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\a': // bell
			if d.BellFunc != nil {
				d.BellFunc()
			}
		case '\b': // backspace
			// whatever is connected to the terminal needs to handle line/character editing
			// however, when the terminal gets a backspace, that's the same as just moving cursor
			// one space to the left. To perform a what looks like an actual backspace you must
			// send "\b \b".
			d.cursor.col = max(d.cursor.col-1, 0)
		case '\t': // tab
			// move cursor to nearest multiple of TabSize, but don't move to next row
			d.cursor.col = min(d.cols-2, d.cursor.col+d.Config.TabSize-(d.cursor.col%d.Config.TabSize))
		case '\r': // carriage return
			d.cursor.col = 0
		case '\n': // linefeed
			// if scroll region is not the whole screen, trying to do a new line past the end
			// of the last row should be treated as a carriage return
			d.cursor.col = 0
			if d.cursor.row == d.scrollRegion[1] {
				d.Scroll(1)
				continue
			}
			if d.cursor.row < d.rows-1 {
				d.cursor.row++
			}
		case 0x0E: // shift out (use alt character set)
			d.Render.active.shift = 1
			d.updateAttr()
		case 0x0F: // shift in (use regular char set)
			d.Render.active.shift = 0
			d.updateAttr()
		case 0x1b: // ESC aka ^[
			n, err = consumeEscSequence(runes[i:])
			if err != nil {
				// copy runes[i:] to d.inputBuf and wait for more input
				d.inputBuf = runes[i:]
				i += len(runes[i:])
				continue
			}
			d.HandleEscSequence(runes[i : i+n])
			i += n - 1
		default:
			// if we're past the end of the screen (remember, d.cols=number of columns but cursor.col is 0 indexed)
			if d.cursor.col == d.cols {
				// back to the beginning
				d.cursor.col = 0
				// scroll if necessary otherwise just move on to the next row
				if d.cursor.row == d.scrollRegion[1] {
					d.Scroll(1)
				} else if d.cursor.row < d.rows-1 {
					d.cursor.row++
				}
			}
			// render our single rune
			d.RenderRune(runes[i])
			// if we drew something the screen, we increment the column.
			d.cursor.col++
		}
	}

	return len(data), nil
}

func (d *Device) Scroll(rowAmount int) {
	// scrollArea Empty means scroll the whole screen--we can use more efficient algos for that
	if d.scrollArea.Empty() {
		// if the underlying image supports Scroll(), use that
		// if scrollable, ok := d.Render.Image.(gfx.Scroller); ok {
		// scrollable.Scroll(rowAmount * d.Render.cell.Dy())
		if scrollable, ok := d.Render.Image.(gfx.RegionScroller); ok {
			scrollable.RegionScroll(d.Render.Bounds(), rowAmount*d.Render.cell.Dy())
		} else {
			// use softscroll
			// probably adding a bug here related to xform.Translate
			softRegionScroll(d.Render.Image, d.Render.Bounds(), rowAmount*d.Render.cell.Dy())
		}

		// fill in scrolls section with background
		if rowAmount > 0 {
			d.Clear(0, d.rows-rowAmount, d.cols, d.rows)
		} else {
			d.Clear(0, 0, d.cols, -rowAmount)
		}

		return
	}

	// scrollArea is set; must scroll a subsection
	if scrollable, ok := d.Render.Image.(gfx.RegionScroller); ok {
		scrollable.RegionScroll(d.scrollArea, rowAmount*d.Render.cell.Dy())
	} else {
		// underlaying image doesn't support gfx.RegionScroller; use softRegionScroll
		softRegionScroll(d.Render.Image, d.scrollArea, rowAmount*d.Render.cell.Dy())
	}

	// fill in scrolls section with background
	if rowAmount > 0 {
		d.Clear(0, d.scrollRegion[1]-rowAmount+1, d.cols, d.scrollRegion[1]+1)
	} else {
		d.Clear(0, d.scrollRegion[0], d.cols, d.scrollRegion[0]-rowAmount)
	}
}

// should probably move to gfx package
func softRegionScroll(img draw.Image, region image.Rectangle, amount int) {
	softVectorScroll(img, region, image.Pt(0, amount))
}

func softVectorScroll(img draw.Image, region image.Rectangle, vector image.Point) {
	region = img.Bounds().Intersect(region)
	var dst, src image.Point
	for y := range region.Dy() {
		for x := range region.Dx() {
			dst.X, dst.Y = region.Min.X+x, region.Min.Y+y
			if vector.Y < 0 {
				dst.Y = region.Min.Y + region.Dy() - (y + 1)
			}
			if vector.X < 0 {
				dst.X = region.Min.X + region.Dx() - (x + 1)
			}
			src = dst.Add(vector).Mod(region)
			img.Set(dst.X, dst.Y, img.At(src.X, src.Y))
		}
	}

	return
}

// ColsRemaining returns how many columns are remaining until EOL
func (d *Device) ColsRemaining() int {
	return d.cols - d.cursor.col
}

func (d *Device) MoveCursorRel(x, y int) {
	d.cursor.col = bound(x+d.cursor.col, 0, d.cols-1)
	d.cursor.row = bound(y+d.cursor.row, 0, d.rows-1)
}

func (d *Device) MoveCursorAbs(x, y int) {
	d.cursor.col = bound(x, 0, d.cols-1)
	d.cursor.row = bound(y, 0, d.rows-1)
}

func (d *Device) setScrollRegion(start, end int) {
	d.scrollArea.Min.X = d.Render.Bounds().Min.X
	d.scrollArea.Max.X = d.Render.Bounds().Max.X

	d.scrollRegion[0] = bound((start - 1), 0, d.rows-1)
	d.scrollRegion[1] = bound((end - 1), 0, d.rows-1)

	d.scrollArea.Min.Y = (d.scrollRegion[0] * d.Render.cell.Dy())
	// + 1 because internally we are 0-indexed, but ANSI escape codes are 1-indexed
	// + another 1 because we want the bottom of the nth cell, not the top
	d.scrollArea.Max.Y = (d.scrollRegion[1] + 1) * d.Render.cell.Dy()

	// honor our offset
	d.scrollArea = d.scrollArea.Add(d.Render.bounds.Min)
	//draw.Draw(d.Render, d.scrollArea, xform.InvertColors(d.Render), d.scrollArea.Min, draw.Src)

	// if you mess up setting the scroll area, just forget the whole thing.
	if (start == 0 && end == 0) || start >= end || d.scrollArea.Eq(d.Render.Bounds()) {
		d.scrollArea = image.Rectangle{}
		d.scrollRegion = [2]int{0, d.rows - 1}
	}
}
