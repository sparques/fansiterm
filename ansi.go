package fansiterm

// ansi.go is largely just an implementation of https://en.wikipedia.org/wiki/ANSI_escape_code
// TODO Actually Implement all of ANSI X3.64

import (
	"bytes"
	"encoding/base64"
	"errors"
	"image"
	"strings"

	"golang.org/x/exp/constraints"
)

var errEscapeSequenceIncomplete = errors.New("escape sequence incomplete")

var (
	// ShowEsc if set to true (default false) prints to stdout escape sequences as received by fansiterm
	ShowEsc bool
	// ShowUnhandled if set to true (default false) prints to stdout escape sequencies that fansiterm does not actually handle.
	ShowUnhandled bool
)

// HandleEscSequence handles escape sequences. This should be the whole complete
// sequence. Bounds are not checked so an incomplete sequence will cause
// a panic.
func (d *Device) handleEscSequence(seq []byte) {
	if ShowEsc {
		log.Info("handling escape sequence", "sequence", string(seq))
	}
	switch seq[0] {
	case '7': // save cursor position
		d.cursor.prevPos[0] = d.cursor.col
		d.cursor.prevPos[1] = d.cursor.row
	case '8': // restore cursor position
		d.cursor.col = d.cursor.prevPos[0]
		d.cursor.row = d.cursor.prevPos[1]
	case 'c': // reset
		d.Reset()
	// case '#': // ESC#8 "Confidence Test"
	// 	d.cursor.MoveAbs(0, 0)
	// 	// abuse inputBuf...
	// 	d.inputBuf = append(d.inputBuf, slices.Repeat([]rune{'E'}, d.rows*d.cols)...)
	case '[':
		d.handleCSISequence(seq[1:])
	case ']':
		d.handleOSCSequence(seq[1:])
	case 'M': // Move cursor up; if at top of screen, scroll up one line
		if d.cursor.row == 0 {
			d.Scroll(-1)
		} else {
			d.cursor.row--
		}
	case '(': // set G0
		switch seq[1] {
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
		switch seq[1] {
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
		d.handleFansiSequence(seq[1:])
	case '>': // auxilary keypad numeric mode
		fallthrough
	case '=': // auxilary keypad application mode
		fallthrough
	default:
		if ShowUnhandled {
			log.Warn("unhandled escape sequence", "sequence", string(seq))
		}
	}
	d.updateAttr()
}

// consumeEscSequence figures out where the escape sequence in data ends.
// It assumes data[0] is the first byte *after* 0x1b.
func consumeEscSequence(data []byte) (n int, err error) {
	if len(data) < 1 {
		// need more bytes
		return 0, errEscapeSequenceIncomplete
	}
	switch data[0] {
	case 'X', ']', 'P', '/': // SOS, OSC, DCS, and my own private sequence
		// For Start of String, Operating System Command, and Device Control String, read
		// until we encounter String Terminator, ESC\
		for n = 1; n < len(data); n++ {
			// handle ESC]R
			if n == 1 && data[n] == 'R' && data[n-1] == ']' {
				return n + 1, nil
			}
			if data[n] == '\a' || (data[n-1] == 0x1b && data[n] == '\\') {
				return n + 1, nil
			}
		}
	case '[': // CSI
		for n = 1; n < len(data); n++ {
			if data[n] >= 0x40 {
				return n + 1, nil
			}
		}
		return 0, errEscapeSequenceIncomplete
	case '(', ')':
		if len(data) < 2 {
			return 0, errEscapeSequenceIncomplete
		}
		// ESC(0 for line drawing
		// ESC(B for regular
		return 2, nil
	default:
		// Unsupported escape sequence, just skip it?
		return 1, nil
	}

	// got to here? need more data
	return 0, errEscapeSequenceIncomplete
}

// getNumericArgs beaks apart seq at ';' characters and then tries to convert
// each piece into an integer. If it fails to convert, def is used.
func getNumericArgs(seq []byte, def int) (args []int) {
	return appendNumericArgs(args, seq, def)
}

func appendNumericArgs(dst []int, seq []byte, def int) []int {
	if len(seq) == 0 {
		return append(dst, def)
	}

	start := 0
	for i := 0; i <= len(seq); i++ {
		if i < len(seq) && seq[i] != ';' {
			continue
		}
		dst = append(dst, parseNumericArg(seq[start:i], def))
		start = i + 1
	}

	return dst
}

func parseNumericArg(seq []byte, def int) int {
	if len(seq) == 0 {
		return def
	}

	sign := 1
	i := 0
	switch seq[0] {
	case '-':
		sign = -1
		i = 1
	case '+':
		i = 1
	}
	if i == len(seq) {
		return def
	}

	n := 0
	for ; i < len(seq); i++ {
		r := seq[i]
		if r < '0' || r > '9' {
			return def
		}
		n = n*10 + int(r-'0')
	}
	return sign * n
}

func bound[N constraints.Integer](x, minimum, maximum N) N {
	return min(max(x, minimum), maximum)
}

func trimST(seq []byte) []byte {
	if len(seq) == 0 {
		return seq
	}
	switch {
	case seq[len(seq)-1] == '\a':
		return seq[:len(seq)-1]
	case len(seq) >= 2 && seq[len(seq)-2] == 0x1b && seq[len(seq)-1] == '\\':
		return seq[:len(seq)-2]
	default:
		return seq
	}
}

// DecodeImageData accepts base64 encoded data and attempts to
// decode it as an image, returning the image.
func DecodeImageData(data []byte) (image.Image, error) {
	pixData, err := base64.StdEncoding.DecodeString(string(data))
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewBuffer(pixData))

	return img, err
}

func splitParams(data []byte) (split [][]byte) {
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

func parsePoint(seq []byte) (pt image.Point, ok bool) {
	x, n, ok := parseIntPrefix(seq)
	if !ok || n >= len(seq) || seq[n] != ',' {
		return image.Point{}, false
	}
	y, m, ok := parseIntPrefix(seq[n+1:])
	if !ok {
		return image.Point{}, false
	}
	if n+1+m != len(seq) {
		return image.Point{}, false
	}
	return image.Pt(x, y), true
}

func parseRect(seq []byte) (rect image.Rectangle, ok bool) {
	var n int
	rect.Min.X, n, ok = parseIntPrefix(seq)
	if !ok || n >= len(seq) || seq[n] != ',' {
		return image.Rectangle{}, false
	}
	seq = seq[n+1:]
	rect.Min.Y, n, ok = parseIntPrefix(seq)
	if !ok {
		return image.Rectangle{}, false
	}
	seq = seq[n:]
	if len(seq) == 0 || seq[0] != ';' {
		return image.Rectangle{}, false
	}
	seq = seq[1:]
	rect.Max.X, n, ok = parseIntPrefix(seq)
	if !ok || n >= len(seq) || seq[n] != ',' {
		return image.Rectangle{}, false
	}
	seq = seq[n+1:]
	rect.Max.Y, n, ok = parseIntPrefix(seq)
	if !ok {
		return image.Rectangle{}, false
	}
	if n != len(seq) {
		return image.Rectangle{}, false
	}
	return rect, true
}

func parseIntPrefix(seq []byte) (value int, consumed int, ok bool) {
	if len(seq) == 0 {
		return 0, 0, false
	}
	sign := 1
	switch seq[0] {
	case '-':
		sign = -1
		consumed = 1
	case '+':
		consumed = 1
	}
	if consumed == len(seq) || seq[consumed] < '0' || seq[consumed] > '9' {
		return 0, 0, false
	}
	for consumed < len(seq) {
		r := seq[consumed]
		if r < '0' || r > '9' {
			break
		}
		value = value*10 + int(r-'0')
		consumed++
	}
	return sign * value, consumed, true
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
