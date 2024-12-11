package fansiterm

// ansi.go is largely just an implementation of https://en.wikipedia.org/wiki/ANSI_escape_code

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"image"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

var errEscapeSequenceIncomplete = errors.New("escape sequence incomplete")

var (
	ShowEsc       bool
	ShowUnhandled bool
)

// consumeEscSequence figures out where the escape sequence in data ends.
// It assumes data[0] == 0x1b.
func consumeEscSequence(data []rune) (n int, err error) {
	if len(data) < 2 {
		// need more bytes
		return 0, errEscapeSequenceIncomplete
	}
	switch data[1] {
	case 'X', ']', 'P', '/': // SOS, OSC, DCS, and my own private sequence
		// For Start of String, Operating System Command, and Device Control String, read
		// until we encounter String Terminator, ESC\
		for n = 1; n < len(data); n++ {
			// handle ESC]R
			if n == 2 && data[n] == 'R' && data[n-1] == ']' {
				return n + 2, nil
			}
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
	case '(', ')':
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
	for _, arg := range splitParams(seq) {
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
	if ShowEsc {
		fmt.Println(seqString(seq))
	}
	switch seq[1] {
	case '7': // save cursor position
		d.cursor.prevPos[0] = d.cursor.col
		d.cursor.prevPos[1] = d.cursor.row
	case '8': // restore cursor position
		d.cursor.col = d.cursor.prevPos[0]
		d.cursor.row = d.cursor.prevPos[1]
	case 'c': // reset
		d.Reset()
	case '[':
		d.HandleCSISequence(seq[2:])
	case ']':
		d.HandleOSCSequence(seq[2:])
	case 'M': // Move cursor up; if at top of screen, scroll up one line
		if d.cursor.row == 0 {
			d.Scroll(-1)
		} else {
			d.cursor.row--
		}
	case '(': // set G0
		switch seq[2] {
		case '0':
			// d.Render.G0 = d.Render.AltCharSet
			d.Render.active.g[0] = &d.Render.AltCharSet
		case 'B':
			fallthrough
		default:
			// d.Render.G0 = d.Render.CharSet
			d.Render.active.g[0] = &d.Render.CharSet
		}
	case ')': // set G1
		// B for regular, 0 for line drawing
		switch seq[2] {
		case '0':
			// d.Render.G1 = d.Render.AltCharSet
			d.Render.active.g[1] = &d.Render.AltCharSet
		case 'B':
			fallthrough
		default:
			// d.Render.G1 = d.Render.CharSet
			d.Render.active.g[1] = &d.Render.CharSet
		}
	case '/':
		d.HandleFansiSequence(seq[2:])
	case '>': // auxilary keypad numeric mode
		fallthrough
	case '=': // auxilary keypad application mode
		fallthrough
	default:
		if ShowUnhandled {
			fmt.Println("Unhandled ESC:", seqString(seq))
		}
	}
	d.updateAttr()
}

func trimST(seq []rune) []rune {
	switch {
	case seq[len(seq)-1] == '\a':
		return seq[:len(seq)-1]
	case seq[len(seq)-2] == 0x1b && seq[len(seq)-1] == '\\':
		return seq[:len(seq)-2]
	default:
		return seq
	}
}

func DecodeImageData(data []rune) (image.Image, error) {
	pixData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewBuffer(pixData))

	return img, err
}

func splitParams(data []rune) (split [][]rune) {
	prev := 0
	for i := range data {
		if data[i] == ';' {
			split = append(split, data[prev:i])
			prev = i + 1
		}
	}

	split = append(split, data[prev:])

	return
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
