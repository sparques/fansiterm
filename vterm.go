package fansiterm

import (
	"bytes"
	"image"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/inconsolata"
)

/*
vterm implements a virtual terminal. It supports being io.Read() from and io.Write()n to. It handles the cursor and processing of
escape sequences.
*/

type Device struct {
	// BellFunc is called if it is non-null and the terminal would
	// display a bell character
	BellFunc func()
	Config   Config

	cols, rows int
	cursorPos  int
	// showCursor is whether or not we're supposed to be showing the cursor
	showCursor bool
	// curosrVisible is whether the curosr is currently displayed
	cursorVisible bool

	// attr tracks the currently applied attributes
	attr Attr

	// attrDefault is used when attr is zero-value or nil
	attrDefault Attr

	buf    draw.Image
	offset image.Point

	fontDraw    font.Drawer
	fontDescent int
	// add Regular, Bold, Italic font.Face entries?
	useAltCharSet bool

	altCharSet TileSet

	cell image.Rectangle
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
	// consider using an image.Uniform instead of a color.Color; the color is still accessible
	// as a sub-field of image.Uniform, but also you don't ever have to instantiate an image.Uniform
	// when you need to draw something
	Fg Color
	Bg Color
}

var AttrDefault = Attr{
	Fg: ColorWhite,
	Bg: ColorBlack,
}

var ConfigDefault = Config{
	TabSize:     8,
	CursorStyle: CursorBlock,
	CursorBlink: true,
}

// New returns an initialized *Device. If buf is nil, an internal buffer is used. Otherwise
// if you specify a hardware backed draw.Image, writes to Device will immediately be written
// to the backing hardware--whether this is instaneous or buffered is up to the device and the
// device driver.
func New(cols, rows int, buf draw.Image) *Device {
	// Eventually I'd like to support different fonts and dynamic resizing
	// I'm trying to get to an MVP first.
	// thus, hardcoded font face
	fontFace := inconsolata.Regular8x16
	// 7x13 is smaller and non-antialiased. For small screens it might be a better choice
	// than the 8x13 pre-render of inconsolata, however it doesn't have as many unicode-glyps
	// as inconsolata.
	//fontFace := basicfont.Face7x13
	cell := image.Rect(0, 0, fontFace.Advance, fontFace.Height)

	if buf == nil {
		buf = image.NewRGBA(image.Rect(0, 0, cols*cell.Max.X, rows*cell.Max.Y))
	}

	draw.Draw(buf, buf.Bounds(), image.Black, image.Point{}, draw.Src)

	return &Device{
		cols:       cols,
		rows:       rows,
		attr:       AttrDefault,
		buf:        buf,
		cell:       cell,
		showCursor: true,
		fontDraw: font.Drawer{
			Dst:  buf,
			Face: fontFace,
		},
		fontDescent: fontFace.Descent,
		attrDefault: AttrDefault,
		Config:      ConfigDefault,
		altCharSet:  NewTileSet(),
	}
}

// NewAtResolution is like New, but rather than specifying the columns and rows,
// you specify the desired resolution. The maximum rows and cols will be determined
// automatically and the terminal rendered in the center.
// TODO: allow offset to be manually specified
func NewAtResolution(x, y int, buf draw.Image) *Device {
	// TODO: This is a crappy way of figuring out what font we're using. Do something else.
	d := New(1, 1, nil)
	// use d.cell to figure out rows and cols; integer division will round down
	// which is what we want
	cols := x / d.cell.Max.X
	rows := y / d.cell.Max.Y
	offset := image.Pt((x%d.cell.Max.X)/2, (y%d.cell.Max.Y)/2)

	//fmt.Println("Res:", x, "x", y, "Cols:", cols, "Rows:", rows, "Offset:", offset)

	if buf == nil {
		buf = image.NewRGBA(image.Rect(0, 0, x, y))
	}

	draw.Draw(buf, buf.Bounds(), image.Black, image.Point{}, draw.Src)

	return New(cols, rows, NewImageTranslate(offset, buf))

}

/* Broken: need to make work with cursor display
func (d *Device) WriteAt(p []byte, off int64) (n int, err error) {
	oldPos := d.cursorPos
	defer func() {
		d.cursorPos = oldPos
	}()
	if off < int64(d.rows*d.cols) {
		d.cursorPos = int(off)
	} else {
		d.cursorPos = d.rows * d.cols
	}
	return d.Write(p)
}
*/

func isControl(r rune) bool {
	return r < 0x20
}

func (d *Device) Write(data []byte) (n int, err error) {
	// TODO: batch together runs of bytes that don't have escape
	// characters in them

	runes := bytes.Runes(data)

	// first un-invert cursor (if we're showing it)
	if d.cursorVisible {
		d.toggleCursor()
	}

	// Safe shortcut? If no control characters, render all
	// Disabled because need to fix RenderRune -> RenderRunes
	/*
		if len(runes) <= d.ColsRemaing() && !bytes.ContainsFunc(data, isControl) {
				d.RenderBytes(data)
			}
	*/
	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\a': // bell
			if d.BellFunc != nil {
				d.BellFunc()
			}
		// case '\b': // do I need to handle backspace here?!
		case '\t': // tab
			d.cursorPos += d.Config.TabSize - (d.cursorPos%d.cols)%d.Config.TabSize
		case '\r': // carriage return
			d.cursorPos -= d.cursorPos % d.cols
		case '\n': // linefeed
			d.cursorPos += d.cols - (d.cursorPos % d.cols)
			d.ScrollToCursor()
		case 0x0E: // shift out (use alt character set)
			d.useAltCharSet = true
		case 0x0F: // shift in (use regular char set)
			d.useAltCharSet = false
		case 0x1b: // ESC aka ^[
			// at least one more byte available?
			if i >= len(runes) {
				break
			}
			i++
			if runes[i] == '[' {
				i++
				start := i
				for i < len(runes) && runes[i] < 0x40 {
					i++
				}
				// TODO: Is there a better way to pass this?
				// If I switch HandleEscSequence() to use []rune instead of []byte
				// I can avoid the conversion, but does that actually make things better?
				d.HandleEscSequence([]byte(string(runes[start : i+1])))
			}
		default:
			// write character to buf
			d.RenderRune(runes[i])
			d.MoveCursorRight()
		}
	}

	// finally, update cursor, if needed
	if d.showCursor {
		d.toggleCursor()
	}

	return len(data), nil
}

func (d *Device) Scroll(amount int) {
	if amount > 0 {
		//shift the lower portion of the image up
		draw.Draw(d.buf,
			image.Rect(0, 0, d.cols*d.cell.Max.X, (d.rows-amount)*d.cell.Max.Y),
			d.buf,
			image.Pt(0, amount*d.cell.Max.Y),
			draw.Src)
		// fill in the lower portion with Bg
		draw.Draw(d.buf,
			image.Rect(0, (d.rows-amount)*d.cell.Max.Y, d.cols*d.cell.Max.X, d.rows*d.cell.Max.Y),
			&image.Uniform{d.attr.Bg},
			image.Pt(0, amount*d.cell.Max.Y),
			draw.Src)
		return
	}
	amount = -amount
}

func (d *Device) CursorCol() int {
	return d.cursorPos % d.cols
}

func (d *Device) CursorRow() int {
	return d.cursorPos / d.cols
}

// ColsRemaining returns how many columns are remaining until EOL
func (d *Device) ColsRemaining() int {
	return d.cols - (d.cursorPos % d.cols)
}

func (d *Device) MoveCursorRight() {
	d.cursorPos++

	// if we're at the end of the vterm, scroll down one line
	d.ScrollToCursor()
}

func (d *Device) MoveCursorRel(x, y int) {
	x = bound(x, -(d.cursorPos % d.cols), d.cols-(d.cursorPos%d.cols))
	y = bound(y, -(d.cursorPos / d.cols), d.rows-(d.cursorPos/d.cols))

	d.cursorPos += y*d.cols + x
}

func (d *Device) MoveCursorAbs(x, y int) {
	x = bound(x, 0, d.cols)
	y = bound(y, 0, d.rows)

	d.cursorPos = y*d.cols + x
}

func (d *Device) ScrollToCursor() {
	if d.cursorPos >= d.rows*d.cols {
		//d.cursorPos -= d.cols
		d.cursorPos = d.cols * (d.rows - 1)
		d.Scroll(1)
	}
}

func (d *Device) offsetToRowCol(off int) (row, col int) {
	return off / d.cols, off % d.cols
}
