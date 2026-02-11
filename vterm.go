// Package fansiterm provides a fully featured, graphics-capable virtual terminal implementation
// designed for environments with minimal or no operating support.
// Originally implemented to run on microcontrollers and works
// well with TinyGo.
package fansiterm

import (
	"bytes"
	"image"
	"image/color"
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

// Device implements a virtual terminal. It satisfies the io.Writer interface
// and supports ANSI escape sequences, custom fonts, raster graphics, and more.
// It is thread-safe and optimized for embedded or minimal environments.
//
//go:export
type Device struct {
	// BellFunc is called if it is non-null and the terminal would
	// display a bell character
	BellFunc func(id string)

	// Config specifies the runtime configurable features of fansiterm.
	Config Config

	// ConfigUpdate, if non-nil, is called when the config changes.
	ConfigUpdate func(conf Config)
	// Consider adding a ConfigSet func(Config) field. Just sayin'

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

	// saveBuf is used to store the main buffer when the alternate screen
	// is used.
	saveBuf draw.Image

	// Output specifies the program attached to the terminal. This should be the
	// same interface that the input mechanism (whatever that may be) uses to write
	// to the program. On POSIX systems, this would be equivalent to Stdin.
	// Default is io.Discard. Setting to nil will cause Escape Sequences that
	// write a response to panic.
	Output io.Writer

	// UserResetFunc is called when fansiterm's Reset() method is called. This
	// is the same Reset triggered by an \x1bc escape sequence. This can be used
	// to reset a hardware display.
	UserResetFunc func()

	// writeQueue is a channel for queuing up writes. On large systems, this is
	// not necessary. But in constrained spaces like microcontrollers, Write is
	// likely being called from an interrupt service routine and needs to return
	// as quickly as possible. So Write() copies data and then puts it into
	// the writeQueue
	// Writing to a chan in an ISR isn't great either, but the chan should be
	// sized such that it never blocks.
	writeQueue chan []byte

	done chan struct{}

	sync.Mutex
}

// Attr defines the attributes applied to rendered text.
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

// New initializes a new terminal device with the specified dimensions and optional draw.Image buffer.
// If buf is nil, a default in-memory RGBA buffer is allocated. The terminal's character size is fixed.
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
	//colorSystem := NewColorSystem(buf.ColorModel())

	// figure out our actual terminal bounds.
	bounds := image.Rect(0, 0, cell.Dx()*cols, cell.Dy()*rows).Add(buf.Bounds().Min)

	// if our backing buffer is bigger than our grid of cells, center the terminal
	// ... more or less.

	// figure out how much we need to shift around
	offset := image.Pt((buf.Bounds().Dx()%cell.Dx())/2, (buf.Bounds().Dy()%cell.Dy())/2)

	// shift around
	bounds = bounds.Add(offset)

	charSet := tiles.NewMultiTileSet(sweet16.Regular8x16, drawing.TileSet)
	altCharSet := altCharsetViaUnicode(charSet)

	d := &Device{
		writeQueue: make(chan []byte, 256),
		done:       make(chan struct{}),
		// bufChan:    make(chan draw.Image),
		cols: cols,
		rows: rows,
		Render: Render{
			Image:         buf,
			bounds:        bounds,
			AltCharSet:    altCharSet,
			CharSet:       charSet,
			BoldCharSet:   sweet16.Bold8x16,
			ItalicCharSet: &tiles.Italics{Tiler: charSet},
			cell:          cell,
			cursorFunc:    blockRect,
		},
		cursor: Cursor{
			show: true,
		},
		Config:        NewConfig(),
		Output:        io.Discard,
		scrollRegion:  [2]int{0, rows - 1},
		UserResetFunc: func() {},
	}

	// link cursor's rows/cols back to *Device
	d.cursor.rows = &d.rows
	d.cursor.cols = &d.cols

	// Establish defaults
	d.attrDefault.Fg = defaultFg
	d.attrDefault.Bg = defaultBg

	d.useBuf(buf)

	d.Render.active.g = make([]*tiles.Tiler, 2)
	d.Reset()
	d.updateAttr()

	go d.queueHandler()

	return d
}

// NewAtResolution returns a new Device sized to fit a resolution (x,y), centering the terminal.
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

func (d *Device) UseBuf(buf draw.Image) {
	d.preUpdate()
	d.useBuf(buf)
	d.postUpdate()
}

func (d *Device) useBuf(buf draw.Image) {
	cell := d.Render.cell
	d.cols = buf.Bounds().Dx() / 8
	d.rows = buf.Bounds().Dy() / 16

	// save the old buf
	origBuf := copyImage(d.Render.Image)

	// figure out our actual terminal bounds.
	bounds := image.Rect(0, 0, cell.Dx()*d.cols, cell.Dy()*d.rows).Add(buf.Bounds().Min)

	// if our backing buffer is bigger than our grid of cells, center the terminal
	// ... more or less.

	// figure out how much we need to shift around
	offset := image.Pt((buf.Bounds().Dx()%cell.Dx())/2, (buf.Bounds().Dy()%cell.Dy())/2)

	// shift around
	bounds = bounds.Add(offset)
	d.Render.bounds = bounds
	d.Render.Image = buf

	// use hardware accelerated functions where possible
	// VectorScroll is the most flexible and least performant, even if implemented in hardware.
	// VectorScroll can be used to perform RegionScroll and Scroll
	// if the underlaying driver does not support RegionScroll or
	// Scroll. We use a priority fallback order:
	// First, use driver supported VectorScroll otherwise use software
	// Use driver supported RegionScroll otherwise use VectorScroll
	// Use driver supported Scroll if supported, otherwise fall back
	// to RegionScroll.
	if scrollable, ok := d.Render.Image.(gfx.VectorScroller); ok {
		d.Render.vectorScroll = scrollable.VectorScroll
	} else {
		d.Render.vectorScroll = func(r image.Rectangle, v image.Point) { softVectorScroll(d.Render.Image, r, v) }
	}

	if scrollable, ok := d.Render.Image.(gfx.RegionScroller); ok && offset.X == 0 {
		d.Render.regionScroll = scrollable.RegionScroll
	} else {
		d.Render.regionScroll = func(region image.Rectangle, pixAmt int) {
			d.Render.vectorScroll(region, image.Pt(0, pixAmt))
		}
	}

	// we can only use hardware scroll if fansi term is using the whole
	// screen, otherwise we need to do a region scroll or vector scroll
	switch {
	case offset == image.Point{}:
		// offset is zero, we can use Scroll or RegionScroll
		if scrollable, ok := d.Render.Image.(gfx.Scroller); ok && d.Render.Bounds().Eq(buf.Bounds()) {
			d.Render.scroll = scrollable.Scroll
		} else {
			// fall back on vectorScroll, be it software or hardware
			d.Render.scroll = func(pixAmt int) {
				d.Render.regionScroll(d.Render.bounds, pixAmt)
			}
		}
	case offset.X == 0:
		// offset only exists for Y, can use RegionScroll
		d.Render.scroll = func(pixAmt int) {
			d.Render.regionScroll(d.Render.bounds, pixAmt)
		}
	default:
		// there's a X and Y offset, must use VectorScroll
		d.Render.scroll = func(pixAmt int) {
			d.Render.vectorScroll(d.Render.bounds, image.Pt(0, pixAmt))
		}
	}

	if fillable, ok := d.Render.Image.(gfx.Filler); ok {
		d.Render.fill = fillable.Fill
	} else {
		d.Render.fill = func(r image.Rectangle, c color.Color) {
			draw.Draw(d.Render, r, image.NewUniform(c), r.Min, draw.Src)
		}
	}

	d.Render.Fill(d.Render.Image.Bounds(), d.attrDefault.Bg)
	draw.Draw(d.Render.Image, origBuf.Bounds(), origBuf, origBuf.Bounds().Min, draw.Src)

}

func (d *Device) Reset() {
	d.UserResetFunc()
	d.attr = d.attrDefault
	d.Render.active.g[0] = &d.Render.CharSet
	d.Render.active.g[1] = &d.Render.AltCharSet
	d.Render.active.tileSet = d.Render.active.g[0]
	d.clearAll()
	d.cursor.MoveAbs(0, 0)
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

func (d *Device) BlinkCursor() {
	d.toggleCursor()
	if d.Render.DisplayFunc != nil {
		d.Render.DisplayFunc()
	}
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
// character on screen. Offset values are clamped: Negative offset values are
// set to 0, values larger than d.rows * d.cols are set to d.rows*d.cols-1.
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

// GetReader returns an io.Reader that fansiterm will use for output.
// This uses an io.Pipe under the hood. The write portion of the
// pipe displaces (*Device).Output.
// A new pipe is instantiated every time this is called and will
// displace the old pipe.
func (d *Device) GetReader() (rd io.Reader) {
	rd, d.Output = io.Pipe()
	return
}

// Size returns the size of the terminal in rows and columns.
func (d *Device) Size() (int, int) {
	return d.rows, d.cols
}

// Write implements io.Write and is the main way to interract with a (*fansiterm).Device. This is
// essentially writing to the "terminal."
// Writes are more or less unbuffered with the exception of escape sequences. If a partial escape sequence
// is written to Device, the beginning will be bufferred and prepended to the next write.
// Certain broken escape sequence can potentially block forever.
func (d *Device) Write(data []byte) (n int, err error) {
	// this function exists to shorten the amount of code that runs potentially
	// triggered by an interrupt if we're getting data from UART or SPI.
	// Doing a chan write and allocating memory are also not great to do in
	// an interrupt, but we're going with the lesser evils.

	// copy data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)
	// shove into write queue
	d.writeQueue <- dataCopy
	return len(data), nil
}

func (d *Device) Stop() {
	d.done <- struct{}{}
}

// preUpdate runs tasks necessary before updating pixels--e.g. hiding cursor
func (d *Device) preUpdate() {
	// first un-invert cursor (if we're showing it)
	if d.cursor.visible {
		d.toggleCursor()
	}

}

// postUpdate runs tasks necessary after updating pixels--e.g. showing cursor
func (d *Device) postUpdate() {
	if d.Render.DisplayFunc != nil {
		d.Render.DisplayFunc()
	}
}

// write is the actual implementation. Write
func (d *Device) write(data []byte) (n int, err error) {
	d.Lock()

	runes := bytes.Runes(data)

	d.preUpdate()

	if len(d.inputBuf) != 0 {
		runes = append(d.inputBuf, runes...)
		d.inputBuf = []rune{}
	}

	for i := 0; i < len(runes); i++ {
		switch runes[i] {
		case '\a': // bell
			if d.BellFunc != nil {
				d.BellFunc("bel")
			}
		case '\b': // backspace
			// whatever is connected to the terminal needs to handle line/character editing
			// however, when the terminal gets a backspace, that's the same as just moving cursor
			// one space to the left. To perform a what looks like an actual backspace you must
			// send "\b \b".
			d.cursor.col = max(d.cursor.col-1, 0)
		case '\t': // tab
			// move cursor to nearest multiple of TabSize, but don't move to next row
			d.cursor.col = min(d.cols-1, d.cursor.col+d.Config.TabSize-(d.cursor.col%d.Config.TabSize))
		case '\r': // carriage return
			d.cursor.col = 0
		case '\n': // linefeed
			d.cursor.col = 0
			fallthrough
		case '\v', '\f': // vertical tab and form feed (who uses either any more?!)
			// if scroll region is not the whole screen, trying to do a new line past the end
			// of the last row should be treated as a carriage return
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
				break
			}
			d.handleEscSequence(runes[i : i+n])
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
			// Render rune and then
			// increment cursor by width of rune
			// FIXME: corner case where a >1 width rune happens
			// at the last column
			d.cursor.col += d.RenderRune(runes[i])
			if d.Config.Wraparound {
				d.cursor.col = bound(d.cursor.col, 0, d.cols-1)
			}
		}
	}

	// Re-paint cursor if needed
	d.showCursor()

	d.postUpdate()

	d.Unlock()
	return len(data), nil
}

func rectDiff(a, b image.Rectangle) (right, bottom image.Rectangle) {
	a = a.Canon()
	b = b.Canon()

	right.Min.X = min(a.Max.X, b.Max.X)
	// Min.Y is already zero
	right.Max.X = max(a.Max.X, b.Max.X)
	right.Max.Y = max(a.Max.Y, b.Max.Y)

	// bottom.Min.X already zero
	bottom.Min.Y = min(a.Max.Y, b.Max.Y)
	bottom.Max = right.Max

	return
}
