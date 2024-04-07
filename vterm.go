package fansiterm

import (
	"bytes"
	"image"
	"image/draw"
	"io"
	"slices"
	"time"

	"github.com/sparques/fansiterm/tiles"
	"github.com/sparques/fansiterm/tiles/fansi"
	"github.com/sparques/fansiterm/tiles/inconsolata"
	"github.com/sparques/fansiterm/xform"
)

/*
vterm implements a virtual terminal. It supports being io.Write()n to. It handles the cursor and processing of
escape sequences.
*/

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

	// Render collects together all the graphical rendering fields.
	Render Render

	// inputBuf buffers chracters between call writes. This is exclusively used to
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
	offset image.Point
	// TODO: add bold and (maybe) italic? Could try wrapping charSet in a rotateImage(glyph, -5)?
	charSet       tiles.Tiler
	altCharSet    tiles.Tiler
	boldCharSet   tiles.Tiler
	italicCharSet tiles.Tiler
	useAltCharSet bool
	cell          image.Rectangle
	cursorFunc    cursorRectFunc
	// Some displays require a flush / blit / sync call
	// this could be called at the end of (*Device).Write().
	// displayFunc func()
}

const (
	CursorBlock = iota
	CursorBeam
	CursorUnderscore
)

type Config struct {
	TabSize     int
	CursorStyle int
	CursorBlink bool
}

type Attr struct {
	Bold            bool
	Underline       bool
	DoubleUnderline bool
	Strike          bool
	Blink           bool
	Reversed        bool
	Italic          bool
	Fg              Color
	Bg              Color
}

var AttrDefault = Attr{
	Fg: ColorWhite,
	Bg: ColorBlack,
}

var ConfigDefault = Config{
	TabSize: 8,
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

	draw.Draw(buf, buf.Bounds(), image.Black, image.Point{}, draw.Src)

	return &Device{
		cols: cols,
		rows: rows,
		attr: AttrDefault,
		Render: Render{
			Image:         buf,
			altCharSet:    fansi.AltCharSet,
			charSet:       inconsolata.Regular8x16,
			boldCharSet:   inconsolata.Bold8x16,
			italicCharSet: &tiles.Italics{FontTileSet: inconsolata.Regular8x16},
			cell:          cell,
			cursorFunc:    blockRect,
		},
		cursor: Cursor{
			show: true,
		},
		attrDefault: AttrDefault,
		Config:      ConfigDefault,
		Output:      io.Discard,
		Properties:  make(map[Property]string),
	}
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
	offset := image.Pt((x%d.Render.cell.Dx())/2, (y%d.Render.cell.Dy())/2)

	//fmt.Println("Res:", x, "x", y, "Cols:", cols, "Rows:", rows, "Offset:", offset)

	if buf == nil {
		buf = image.NewRGBA(image.Rect(0, 0, x, y))
	}

	draw.Draw(buf, buf.Bounds(), image.Black, image.Point{}, draw.Src)

	if offset.X == 0 && offset.Y == 0 {
		// no offset needed, skip wrapping buf and save us some memory and cycles
		return New(cols, rows, buf)
	} else {
		// return New(cols, rows, xform.Translate(buf, offset))
		// return New(cols, rows, xform.NewImageTranslate(offset, buf))
		return New(cols, rows,
			xform.SubImage(buf, image.Rect(0, 0, cols*d.Render.cell.Dx(), rows*d.Render.cell.Dy()).Add(offset)))
	}

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

	draw.Draw(buf, buf.Bounds(), image.Black, image.Point{}, draw.Src)

	return New(cols, rows, buf)
}

// VisualBell inverts the screen for a quarter second.
func (d *Device) VisualBell() {
	draw.Draw(d.Render, d.Render.Bounds(), xform.InvertColors(d.Render), image.Point{}, draw.Src)
	time.Sleep(time.Second / 4)
	draw.Draw(d.Render, d.Render.Bounds(), xform.InvertColors(d.Render), image.Point{}, draw.Src)
}

// WriteAt works like calling the save cursor position escape sequence, then
// the absolute set cursor position escape sequence, writing to the terminal,
// and then finally restoring cursor position. The offset is just the i'th
// character on screen. Negative offset values are set to 0, values larger than
// d.rows * d.cols are set to d.rows*d.cols.
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
	off = bound(off, 0, int64(d.rows*d.cols))
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

// Write implements io.Write and is the main way to interract with with (*fansiterm).Device. This is
// essentially writing to the "terminal."
// Writes are more or less unbuffered with the exception of escape sequences. If a partial escape sequence
// is written to Device, the beginning will be bufferred and prepended to the next write.
func (d *Device) Write(data []byte) (n int, err error) {
	runes := bytes.Runes(data)

	// first un-invert cursor (if we're showing it)
	if d.cursor.visible {
		d.toggleCursor()
	}

	if len(d.inputBuf) != 0 {
		runes = append(d.inputBuf, runes...)
		d.inputBuf = []rune{}
	}

	var endIdx int
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
			d.cursor.row++
			d.cursor.col = 0
			d.ScrollToCursor()
		case 0x0E: // shift out (use alt character set)
			d.Render.useAltCharSet = true
		case 0x0F: // shift in (use regular char set)
			d.Render.useAltCharSet = false
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
			// consume as many non-control characters as possible
			// render these with RenderRunes
			// increment cursor; increment i

			// Originally I did this with strings.IndexFunc(string(runes[i:]), isControl)
			// however this seems to return the byte offset rather than the rune offset
			endIdx = slices.IndexFunc(runes[i:], isControl)
			if endIdx == -1 {
				endIdx = len(runes[i:])
			}
			// whichever comes first: end of runes, End of row, or a control char
			endIdx = min(len(runes[i:]), d.cols-d.cursor.col, endIdx)
			d.RenderRunes(runes[i : i+endIdx])

			d.cursor.col += endIdx
			if d.cursor.col >= d.cols {
				d.cursor.col = 0
				d.cursor.row++
			}

			i += endIdx - 1
			d.ScrollToCursor()
		}
	}

	// finally, update cursor, if needed
	d.showCursor()

	return len(data), nil
}

func (d *Device) Scroll(amount int) {
	// TODO: add some check here to see if our backing device/buffer supports scrolling
	// TODO: come up with interface for generic scrolling 😂
	if amount > 0 {
		// shift the lower portion of the image up, row by row, starting with the row
		// that will become thew new row zero
		for y := (amount) * d.Render.cell.Dy(); y <= d.Render.cell.Dy()*(d.rows); y++ {
			for x := d.Render.Bounds().Min.X; x <= d.Render.Bounds().Max.X; x++ {
				// if y+-amount*d.Render.cell.Max.Y > d.Render.Bounds().Dy() {
				// 	continue
				// }
				d.Render.Image.Set(x, y-amount*d.Render.cell.Dy()+d.Render.Bounds().Min.Y,
					d.Render.Image.At(x, y+d.Render.Bounds().Min.Y),
				)
			}
		}
		// fill in the lower portion with Bg
		d.Clear(0, d.rows-amount, d.cols, d.rows)
		return
	}

	// negative scrolling
	// shift the upper portion of the image down, pixel line-by-line, starting from bottom
	// use d.Render.cell.Dy() * d.rows instead of d.Render.Bounds().Dy() because if we're using
	// a draw.Image that's wrapped in an imageTranslate, we'll scroll pixes outside our render-area.
	for y := d.Render.cell.Dy()*d.rows + (amount)*d.Render.cell.Dy(); y > 0; y-- {
		for x := d.Render.Bounds().Min.X; x <= d.Render.Bounds().Max.X; x++ {
			if y+-amount*d.Render.cell.Max.Y > d.Render.Bounds().Dy() {
				continue
			}
			d.Render.Image.Set(x, y+-amount*d.Render.cell.Max.Y,
				d.Render.Image.At(x, y),
			)
		}
	}
	// fill in scrolls section with background
	d.Clear(0, 0, d.cols, -amount)
}

// ColsRemaining returns how many columns are remaining until EOL
func (d *Device) ColsRemaining() int {
	return d.cols - d.cursor.col
}

func (d *Device) MoveCursorRel(x, y int) {
	d.cursor.col = bound(x+d.cursor.col, 0, d.cols)
	d.cursor.row = bound(y+d.cursor.row, 0, d.rows)
}

func (d *Device) MoveCursorAbs(x, y int) {
	d.cursor.col = bound(x, 0, d.cols)
	d.cursor.row = bound(y, 0, d.rows)
}

func (d *Device) ScrollToCursor() {
	// this one shouldn't happen
	if d.cursor.col > d.cols {
		d.cursor.col = 0
		d.cursor.row++
	}
	// this is the more common scenario
	if d.cursor.row >= d.rows {
		d.cursor.col = 0
		d.cursor.row = d.rows - 1
		d.Scroll(1)
	}
}
