package fansiterm

import (
	"fmt"
	"image"
	"image/draw"
)

func (d *Device) handleCSISequence(seq []rune) {
	if len(seq) == 0 {
		return
	}
	args := getNumericArgs(seq[:len(seq)-1], 1)
	// last byte of seq tells us what function we're doing
	switch seq[len(seq)-1] {
	case '@': // // Insert Characters. one option numerica arg, default 1
		// TODO really shouldn't be using d.Render / d.Render.Image directly in here.
		// Should have a scroll horizontal function or similar maybe a vectorScroll that works in cells
		curs := d.cursorPt()

		d.Render.VectorScroll(
			image.Rectangle{Min: curs, Max: curs.Add(image.Pt(d.cursor.ColsRemaining()*d.Render.cell.Dx(), d.Render.cell.Dy()))},
			image.Pt(-d.Render.cell.Dx()*args[0], 0))
	case 'A': // Cursor Up, one optional numeric arg, default 1
		d.cursor.MoveRel(0, -args[0])
	case 'B': // Cursor Down, one optional numeric arg, default 1
		d.cursor.MoveRel(0, args[0])
	case 'C': // Cursor Right, one optional numeric arg, default 1
		d.cursor.MoveRel(args[0], 0)
	case 'D': // Cursor Left, one optional numeric arg, default 1
		d.cursor.MoveRel(-args[0], 0)
	case 'E': // Moves cursor to beginning of the line n (default 1) lines down.
		d.cursor.MoveRel(-d.cols, args[0])
	case 'F': // Moves cursor to beginning of the line n (default 1) lines up.
		d.cursor.MoveRel(-d.cols, -args[0])
	case 'G': // Moves the cursor to column n (default 1).
		d.cursor.MoveAbs(args[0]-1, d.cursor.row)
	case 'H', 'f': // Cursor position, Moves the cursor to row n, column m. The values are 1-based, and default to 1 (top left corner) if omitted. A sequence such as CSI ;5H is a synonym for CSI 1;5H as well as CSI 17;H is the same as CSI 17H and CSI 17;1H
		var n, m int = 1, 1
		switch len(args) {
		case 2:
			m = args[1]
			fallthrough
		case 1:
			n = args[0]
		}

		d.cursor.MoveAbs(m-1, n-1)
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
	case 'P': // DCH Delete Character. Delete character(s) to the right of the cursor, shifting as needed
		curs := d.cursorPt()
		d.Render.VectorScroll(
			image.Rectangle{Min: curs, Max: curs.Add(image.Pt(d.cursor.ColsRemaining()*d.Render.cell.Dx(), d.Render.cell.Dy()))},
			image.Pt(d.Render.cell.Dx()*args[0], 0))
		d.Clear(d.cols-args[0], d.cursor.row, d.cols, d.cursor.row+1)

	case 'S': // Scroll whole page up by n (default 1) lines. New lines are added at the bottom.
		d.Scroll(args[0])
	case 'T': // Scroll whole page down by n (default 1) lines. New lines are added at the top.
		d.Scroll(-args[0])
	case 'X': // Delete (clear) cells to the right of the cursor, on the same line
		d.Clear(d.cursor.col, d.cursor.row, bound(args[0]+d.cursor.col, d.cursor.col+1, d.cols), d.cursor.row+1)
	case 'c': // DA Device Attributes
		// Lie and say we're a vt100
		fmt.Fprintf(d.Output, "\x1b[?1;2c")
	case 'd': // CSI n d: Mover cursor to line n
		args = getNumericArgs(seq[:len(seq)-1], 1)
		d.cursor.row = bound(args[0]-1, 0, d.rows)
	case 'm': // CoLoRs!1!! AKA SGR (Select Graphic Rendition)
		args := getNumericArgs(seq[:len(seq)-1], 0)
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case 0:
				d.attr = d.attrDefault
			case 1:
				d.attr.Bold = true
				if d.Config.BoldColors {
					// if BoldColors is enabled, setting bold bumps
					// the fg color
					for i := range 8 {
						if d.attr.Fg == d.Render.colorSystem.PaletteANSI[i] {
							d.attr.Fg = d.Render.colorSystem.PaletteANSI[i+8]
							break
						}
					}
					// don't modify color if we're not using one of the
					// vga colors. There is a corner case of someone using
					// a 24-bit or 256-color color, fixible by using a flag
					// indicating if we've set a ANSI color or not, but...
					// meh.
				}
			case 22:
				d.attr.Bold = false
				if d.Config.BoldColors {
					// if BoldColors is enabled, unsetting bold drops
					// the fg color
					for i := range 8 {
						if d.attr.Fg == d.Render.colorSystem.PaletteANSI[i+8] {
							d.attr.Fg = d.Render.colorSystem.PaletteANSI[i]
							break
						}
					}
					// don't modify color if we're not using one of the
					// vga colors. There is a corner case of someone using
					// a 24-bit or 256-color color, fixible by using a flag
					// indicating if we've set a ANSI color or not, but...
					// meh.
				}
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
			case 8:
				d.attr.Conceal = true
			case 28:
				d.attr.Conceal = false
			case 9:
				d.attr.Strike = true
			// case 10:
			// 	d.Render.active.tileSet = d.Render.G0
			// case 11:
			// 	d.Render.active.tileSet = d.Render.G1
			case 29:
				d.attr.Strike = false
			case 30, 31, 32, 33, 34, 35, 36, 37:
				if d.Config.BoldColors && d.attr.Bold {
					d.attr.Fg = d.Render.colorSystem.PaletteANSI[args[i]-30+8]
				} else {
					d.attr.Fg = d.Render.colorSystem.PaletteANSI[args[i]-30]
				}
			case 39:
				d.attr.Fg = d.attrDefault.Fg
			case 40, 41, 42, 43, 44, 45, 46, 47:
				d.attr.Bg = d.Render.colorSystem.PaletteANSI[args[i]-40]
			case 49:
				d.attr.Bg = d.attrDefault.Bg
			case 90, 91, 92, 93, 94, 95, 96, 97:
				d.attr.Fg = d.Render.colorSystem.PaletteANSI[args[i]-90+8]
			case 100, 101, 102, 103, 104, 105, 106, 107:
				d.attr.Bg = d.Render.colorSystem.PaletteANSI[args[i]-100+8]
			// 24bit True Color and 256-Color support support
			case 38, 48:
				if i+1 >= len(args) {
					continue
				}
				if args[i+1] == 5 {
					// prevent going out of range
					args[i] = args[i] % 256
					if args[i] == 38 {
						d.attr.Fg = d.Render.colorSystem.Palette256[args[i+2]]
					} else {
						d.attr.Bg = d.Render.colorSystem.Palette256[args[i+2]]
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
					d.attr.Fg = d.Render.colorSystem.NewRGB(r, g, b)
				} else {
					d.attr.Bg = d.Render.colorSystem.NewRGB(r, g, b)
				}
			default:
				if ShowUnhandled {
					fmt.Println("Unhandled SGR:", args[i], "(part of", seqString(seq), ")")
				}

			} // switch for SGR

		}
	case 'n': // DSR - Device Status Report
		// args -
		// '5' just returns CSI 0 n
		// '6' return cursor location
		switch args[0] {
		case 5:
			d.Output.Write([]byte{0x1b, '[', '0', 'n'})
		case 6:
			fmt.Fprintf(d.Output, "\x1b[%d;%dR", bound(d.cursor.row+1, 1, d.rows), bound(d.cursor.col+1, 1, d.cols))
		}
	case 'l', 'h': // private on/off extensions
		if seq[0] != '?' || len(seq) < 2 {
			return
		}
		args := getNumericArgs(seq[1:len(seq)-1], 0)
		var set bool
		if seq[len(seq)-1] == 'h' {
			set = true
		}
		switch args[0] {
		case 0, 1: // cursor key mode
			// enable: Application Mode
			// disable: Cursor Mode.
			// This is more an input thing--whatever is writing to fansiterm
			// should check this setting and adjust arrow-key input
			// accordingly.ESC?12h
			d.Config.CursorKeyApplicationMode = set
		case 7: // enable/disable wraparound mode.
			// my god, getting end of line and end of terminal line wrapping
			// working the first place was hard enough.
			// wraparound is the process of if a line over flows (reaches EOL) it should continue onto the next line. With wrap around disabled, once the cursor gets to the end of the line, it no longer advances.
			d.Config.Wraparound = !set
		case 12: // local echo
		case 25: // show/hide cursor
			if set {
				d.cursor.show = true
			} else {
				d.cursor.show = false
				if d.cursor.visible {
					d.toggleCursor()
				}
			}
		case 1000, 1006: // report mouse clicks
		// no, not supported
		case 47, 1049: // alt screen enable/disable
			// 47 is save/restore screen.
			// 1049 is use alternate screen.
			// For fansiterm, there's no difference.
			if !d.Config.AltScreen {
				return
			}
			if set {
				// use AltScreen; just save the buffer
				d.saveBuf = image.NewRGBA(d.Render.bounds)
				draw.Draw(d.saveBuf, d.Render.bounds, d.Render, image.Point{}, draw.Src)
				d.clearAll()
			} else {
				// stop using alt screen, show the saved buffer
				draw.Draw(d.Render, d.Render.bounds, d.saveBuf, image.Point{}, draw.Src)
			}
		case 2004: //bracketed paste enable disable
			// given fansiterm's intended use case, this is going unimplemented.
		default:
			if ShowUnhandled {
				fmt.Println("Unhandled Private Sequence", seqString(seq))
			}
		}
	case 'r': // set scroll region
		if len(args) != 2 {
			return
		}
		d.setScrollRegion(args[0], args[1])
	case 's': // save cursor position
		d.cursor.SavePos()
	case 't':
		switch args[0] {
		case 18: // report terminal size in cells
			fmt.Fprintf(d.Output, "\x1b[8;%d:%dt", d.rows, d.cols)
		case 19: // report terminal size in pixels
			fmt.Fprintf(d.Output, "\x1b[9;%d;%dt", d.Render.Bounds().Dy(), d.Render.Bounds().Dx())
		}
	case 'u': // restore cursor position
		d.cursor.RestorePos()
	default:
		if ShowUnhandled {
			fmt.Println("Unhandled CSI:", seqString(seq))
		}
	} // switch seq[len(seq)-1]
}
