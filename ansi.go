package fansiterm

// ansi.go is largely just an implementation of https://en.wikipedia.org/wiki/ANSI_escape_code

import (
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

var errEscapeSequenceIncomplete = errors.New("escape sequence incomplete")

var ShowUnhandled bool

// consumeEscSequence figures out where the escape sequence in data ends.
// It assumes data[0] == 0x1b.
func consumeEscSequence(data []rune) (n int, err error) {
	if len(data) < 2 {
		// need more bytes
		return 0, errEscapeSequenceIncomplete
	}
	switch data[1] {
	case 'X', ']', 'P': // SOS, OSC, and DCS
		// For Start of String, Operating System Command, and Device Control String, read
		// until we encounter String Terminator, ESC\
		for n = 1; n < len(data); n++ {
			if data[n] == '\a' || (data[n-1] == 0x1b && data[n] == '\\') {
				return n + 1, nil
			}
		}
	case '[': // CSI
		for n = 2; n < len(data); n++ {
			if data[n] >= 0x40 {
				return n + 1, nil
			}
		}
		return 0, errEscapeSequenceIncomplete
	case '(':
		if len(data) < 3 {
			return 0, errEscapeSequenceIncomplete
		}
		// ESC(0 for line drawing
		// ESC(B for regular
		return 3, nil
	default:
		// Unsupported escape sequence, just skip it?
		return 2, nil
	}

	// got to here? need more data
	return 0, errEscapeSequenceIncomplete
}

// getNumericArgs beaks apart seq at ';' characters and then tries to convert
// each piece into an integer. If it fails to convert, def is used.
func getNumericArgs(seq []rune, def int) (args []int) {
	for _, arg := range strings.Split(string(seq), ";") {
		num, err := strconv.Atoi(string(arg))
		if err != nil {
			num = def
		}
		args = append(args, num)
	}
	return args
}

func bound[N constraints.Integer](x, minimum, maximum N) N {
	return min(max(x, minimum), maximum)
}

// HandleEscSequence handles escape sequences. This should be the whole complete
// sequence. Bounds are not checked so an incomplete sequence will cause
// a panic.
func (d *Device) HandleEscSequence(seq []rune) {
	//fmt.Println(seqString(seq))
	switch seq[1] {
	case 'c': // reset
		d.attrDefault = AttrDefault
		d.Render.useAltCharSet = false
		d.clearAll()
		d.MoveCursorAbs(0, 0)
		d.scrollArea = image.Rectangle{}
	case '[':
		d.HandleCSISequence(seq[2:])
	case ']':
		d.HandleOSCSequence(seq[2:])
	case 'M': // scroll up one?
		d.Scroll(-1)
	case '(': // line drawing mode switching
		fallthrough
	case '>': // auxilary keypad numeric mode
		fallthrough
	case '=': // auxilary keypad application mode
		fallthrough
	default:
		if ShowUnhandled {
			fmt.Println("Unhandled ESC:", seqString(seq))
		}
	}
}

func trimST(seq []rune) []rune {
	switch {
	case seq[len(seq)-1] == '\b':
		return seq[:len(seq)-1]
	case seq[len(seq)-2] == 0x1b && seq[len(seq)-1] == '\\':
		return seq[:len(seq)-2]
	default:
		return seq
	}
}

func (d *Device) HandleOSCSequence(seq []rune) {
	seq = trimST(seq)
	if len(seq) == 0 {
		// what does an empty OSC sequence mean?
		// Doing nothing seems safe...
		return
	}
	args := getNumericArgs(seq, 0)
	switch args[0] {
	case 0:
		// xterm set window title
		d.Properties[PropertyWindowTitle] = string(seq[2:])

	default:
		if ShowUnhandled {
			fmt.Println("Unhandled OSC:", seqString(seq))
		}
	}
}

func (d *Device) HandleCSISequence(seq []rune) {
	if len(seq) == 0 {
		return
	}
	args := getNumericArgs(seq[:len(seq)-1], 1)
	// last byte of seq tells us what function we're doing
	switch seq[len(seq)-1] {
	case 'A': // Cursor Up, one optional numeric arg, default 1
		if len(args) == 1 {
			d.MoveCursorRel(0, -args[0])
		}
	case 'B': // Cursor Down, one optional numeric arg, default 1
		if len(args) == 1 {
			d.MoveCursorRel(0, args[0])
		}
	case 'C': // Cursor Right, one optional numeric arg, default 1
		if len(args) == 1 {
			d.MoveCursorRel(args[0], 0)
		}
	case 'D': // Cursor Left, one optional numeric arg, default 1
		if len(args) == 1 {
			d.MoveCursorRel(-args[0], 0)
		}
	case 'E': // Moves cursor to beginning of the line n (default 1) lines down.
		if len(args) == 1 {
			d.MoveCursorRel(-d.cols, args[0])
		}
	case 'F': // Moves cursor to beginning of the line n (default 1) lines up.
		if len(args) == 1 {
			d.MoveCursorRel(-d.cols, -args[0])
		}
	case 'G': // Moves the cursor to column n (default 1).
		if len(args) == 1 {
			d.MoveCursorAbs(args[0], d.cursor.row)
		}
	case 'H': // Cursor position, Moves the cursor to row n, column m. The values are 1-based, and default to 1 (top left corner) if omitted. A sequence such as CSI ;5H is a synonym for CSI 1;5H as well as CSI 17;H is the same as CSI 17H and CSI 17;1H
		var n, m int = 1, 1
		switch len(args) {
		case 2:
			m = args[1]
			fallthrough
		case 1:
			n = args[0]
		}

		d.MoveCursorAbs(m-1, n-1)
	case 'J': // Clears part of the screen. If n is 0 (or missing), clear from cursor to end of screen. If n is 1, clear from cursor to beginning of the screen. If n is 2, clear entire screen (and moves cursor to upper left on DOS ANSI.SYS). If n is 3, clear entire screen and delete all lines saved in the scrollback buffer (this feature was added for xterm and is supported by other terminal applications).
		args = getNumericArgs(seq[:len(seq)-1], 0)
		switch args[0] {
		case 0:
			// clear from cursor to EOL
			d.Clear(d.cursor.col, d.cursor.row, d.cols, d.cursor.row+1)
			// clear area below cursor
			d.Clear(0, d.cursor.row+1, d.cols, d.rows)
		case 1:
			// clear from cursor to beginning of line
			d.Clear(0, d.cursor.row, d.cursor.col, d.cursor.row+1)
			// clear area above cursor
			d.Clear(0, 0, d.cols, d.cursor.row)
		case 2:
			// clear whole screen
			d.Clear(0, 0, d.cols, d.rows)
		}

	case 'K': // Erases part of the line. If n is 0 (or missing), clear from cursor to the end of the line. If n is 1, clear from cursor to beginning of the line. If n is 2, clear entire line. Cursor position does not change.
		args = getNumericArgs(seq[:len(seq)-1], 0)
		switch args[0] {
		case 0:
			// clear from cursor to EOL
			d.Clear(d.cursor.col, d.cursor.row, d.cols, d.cursor.row+1)
		case 1:
			// clear from cursor to beginning of line
			d.Clear(0, d.cursor.row, d.cursor.col, d.cursor.row+1)
		case 2:
			// clear whole line
			d.Clear(0, d.cursor.row, d.cols, d.cursor.row+1)
		}
	case 'L': // Insert Lines
		// save scrollRegion, scrollArea
		r, a := d.scrollRegion, d.scrollArea
		// set scrollRegion to be between cursor and old region's bottom
		d.setScrollRegion(d.cursor.row+1, d.scrollRegion[1]+1)
		// shift down
		d.Scroll(-args[0])
		// clear new lines
		// d.Clear(0, d.scrollRegion[0], d.cols, d.scrollRegion[1]-args[0])
		// restore
		d.scrollRegion, d.scrollArea = r, a
	case 'M': // Delete Line
		// save scrollRegion, scrollArea
		r, a := d.scrollRegion, d.scrollArea
		// set scrollRegion to be between cursor and old region's bottom
		d.setScrollRegion(d.cursor.row+1, d.scrollRegion[1]+1)
		d.Scroll(args[0])
		// d.Clear(0, d.scrollRegion[1]-args[0], d.cols, d.scrollRegion[1])
		// restore
		d.scrollRegion, d.scrollArea = r, a
	case 'P': // DCH Delete Character. Delete character(s) to the right of the cursor
		//args = getNumericArgs(seq[:len(seq)-1], 1)
		// We don't actually track what characters have been typed; we don't support text handling of any kind
		// So this hack of just clearing the whole line to the right of the cursor will have to do.
		// Seems like it works okay so far
		d.Clear(d.cursor.col, d.cursor.row, d.cols, d.cursor.row+1)
	case 'S': // Scroll whole page up by n (default 1) lines. New lines are added at the bottom.
		if len(args) == 1 {
			d.Scroll(args[0])
		}
	case 'T': // Scroll whole page down by n (default 1) lines. New lines are added at the top.
		if len(args) == 1 {
			d.Scroll(args[0])
		}
	case 'c': // DA Device Attributes
		// Lie and say we're a vt100
		fmt.Fprintf(d.Output, "\x1b[?1;2c")
	case 'd': // CSI n d: Mover cursor to line n
		args = getNumericArgs(seq[:len(seq)-1], 0)
		d.cursor.row = bound(args[0]-1, 0, d.rows)
	case 'm': // CoLoRs!1!! AKA SGR (Select Graphic Rendition)
		args := getNumericArgs(seq[:len(seq)-1], 0)
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case 0:
				d.attr = AttrDefault
			case 1:
				d.attr.Bold = true
			case 22:
				d.attr.Bold = false
			case 3:
				d.attr.Italic = true
			case 23:
				d.attr.Italic = false
			case 4:
				d.attr.Underline = true
			case 21:
				d.attr.Underline = true
				d.attr.DoubleUnderline = true
			case 24:
				d.attr.Underline = false
				d.attr.DoubleUnderline = false
			case 5:
				d.attr.Blink = true
			case 25:
				d.attr.Blink = false
			case 7:
				d.attr.Reversed = true
			case 27:
				d.attr.Reversed = false
			case 9:
				d.attr.Strike = true
			case 29:
				d.attr.Strike = false
			case 30:
				d.attr.Fg = ColorBlack
			case 31:
				d.attr.Fg = ColorRed
			case 32:
				d.attr.Fg = ColorGreen
			case 33:
				d.attr.Fg = ColorYellow
			case 34:
				d.attr.Fg = ColorBlue
			case 35:
				d.attr.Fg = ColorMagenta
			case 36:
				d.attr.Fg = ColorCyan
			case 37:
				d.attr.Fg = ColorWhite
			case 40:
				d.attr.Bg = ColorBlack
			case 41:
				d.attr.Bg = ColorRed
			case 42:
				d.attr.Bg = ColorGreen
			case 43:
				d.attr.Bg = ColorYellow
			case 44:
				d.attr.Bg = ColorBlue
			case 45:
				d.attr.Bg = ColorMagenta
			case 46:
				d.attr.Bg = ColorCyan
			case 47:
				d.attr.Bg = ColorWhite
			case 90:
				d.attr.Fg = ColorBrightBlack
			case 91:
				d.attr.Fg = ColorBrightRed
			case 92:
				d.attr.Fg = ColorBrightGreen
			case 93:
				d.attr.Fg = ColorBrightYellow
			case 94:
				d.attr.Fg = ColorBrightBlue
			case 95:
				d.attr.Fg = ColorBrightMagenta
			case 96:
				d.attr.Fg = ColorBrightCyan
			case 97:
				d.attr.Fg = ColorBrightWhite
			case 100:
				d.attr.Bg = ColorBrightBlack
			case 101:
				d.attr.Bg = ColorBrightRed
			case 102:
				d.attr.Bg = ColorBrightGreen
			case 103:
				d.attr.Bg = ColorBrightYellow
			case 104:
				d.attr.Bg = ColorBrightBlue
			case 105:
				d.attr.Bg = ColorBrightMagenta
			case 106:
				d.attr.Bg = ColorBrightCyan
			case 107:
				d.attr.Bg = ColorBrightWhite

			// 24bit True Color and 256-Color support support
			case 38, 48:
				if i+1 >= len(args) {
					continue
				}
				if args[i+1] == 5 {
					// prevent going out of range
					args[i] = args[i] % 256
					if args[i] == 38 {
						d.attr.Fg = Colors256[args[i+2]]
					} else {
						d.attr.Bg = Colors256[args[i+2]]
					}
					i += 2
					continue
				}
				if args[i+1] != 2 {
					continue
				}
				i += 2
				// can proceed
				var r, g, b uint8
				r, g, b = getRGB(args[i:])
				i += 2
				if args[i-4] == 38 {
					d.attr.Fg = NewOpaqueColor(r, g, b)
				} else {
					d.attr.Bg = NewOpaqueColor(r, g, b)
				}

			} // switch for SGR

		}
	case 'n': // DSR - Device Status Report
		// args -
		// '5' just returns CSI 0 n
		// '6' return cursor location
		// Just assume we were passed 6
		switch args[0] {
		case 5:
			d.Output.Write([]byte{0x1b, '[', '0', 'n'})
		case 6:
			fmt.Fprintf(d.Output, "\x1b[%d;%dR", d.cursor.row+1, d.cursor.col+1)
		}
	case 'l', 'h': // on/off extensions
		if seq[0] != '?' || len(seq) < 2 {
			return
		}
		args := getNumericArgs(seq[1:len(seq)-1], 0)
		switch args[0] {
		case 25: // show/hide cursor
			if seq[len(seq)-1] == 'l' {
				d.cursor.show = false
				if d.cursor.visible {
					d.toggleCursor()
				}
			} else {
				d.cursor.show = true
			}
		}
	case 'r': // set scroll region
		if len(args) != 2 {
			return
		}
		d.setScrollRegion(args[0], args[1])
	case 's': // save cursor position
		d.cursor.prevPos[0] = d.cursor.col
		d.cursor.prevPos[1] = d.cursor.row
	case 't':
		switch args[0] {
		case 18: // report terminal size in cells
			fmt.Fprintf(d.Output, "\x1b[8;%d:%dt", d.rows, d.cols)
		case 19: // report terminal size in pixels
			fmt.Fprintf(d.Output, "\x1b[9;%d;%dt", d.Render.Image.Bounds().Dy(), d.Render.Image.Bounds().Dx())
		}
	case 'u': // restore cursor position
		d.cursor.col = d.cursor.prevPos[0]
		d.cursor.row = d.cursor.prevPos[1]
	default:
		if ShowUnhandled {
			fmt.Println("Unhandled CSI:", seqString(seq))
		}
	} // switch seq[len(seq)-1]
}

func getRGB(args []int) (r, g, b uint8) {
	if len(args) > 3 {
		args = args[:3]
	}
	switch len(args) {
	case 0:
		// nothing
	case 3:
		b = uint8(args[2])
		fallthrough
	case 2:
		g = uint8(args[1])
		fallthrough
	case 1:
		r = uint8(args[0])
	}
	return
}

func seqString(seq []rune) string {
	return strings.Map(func(in rune) rune {
		switch in {
		case 0x1b:
			return '⇺'
		default:
			return in
		}
	}, string(seq))
}
