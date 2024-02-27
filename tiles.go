package fansiterm

import (
	"image"
	_ "image/png"
	"io"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// TileSet implements the golang.org/x/image/font.Face interface. It is a simple
// map of rune to image.Image. The images work best as an image.Alpha, that is,
// image data consisting solely of alpha channel.
// TODO: implement variable sized tiles, currently only 8x16 is supported
type TileSet map[rune]image.Image

func NewTileSet() TileSet {
	return make(TileSet)
}

func (ts TileSet) LoadTileFromFile(r rune, file string) {
	fh, err := os.Open(file)
	if err != nil {
		panic(err)
	}
	defer fh.Close()
	ts[r], _, err = image.Decode(fh)
	if err != nil {
		panic(err)
	}
}

func (ts TileSet) LoadTileFromReader(r rune, rd io.Reader) {
	var err error
	ts[r], _, err = image.Decode(rd)
	if err != nil {
		panic(err)
	}
}

func (ts TileSet) Glyph(dot fixed.Point26_6, r rune) (
	dr image.Rectangle, mask image.Image, maskp image.Point, advance fixed.Int26_6, ok bool) {
	glyph, ok := ts[r]

	if !ok {
		return
		// use inconsolata as fallback?
		//return inconsolata.Regular8x16.Glyph(dot, r)
	}

	// Pretty sure I'm doing something wrong here, but tiles/glyphs only lines up if I do this
	// wrong thing here (so there's probably something wrong somewhere else)
	// Fuck if I know.
	bounds := glyph.Bounds().Add(FixedToImagePoint(dot))
	bounds.Min.Y -= glyph.Bounds().Dy()

	return bounds, glyph, image.Point{}, fixed.I(glyph.Bounds().Dx()), ok
}

func (ts TileSet) Close() error {
	return nil
}

func (ts TileSet) GlyphBounds(r rune) (bounds fixed.Rectangle26_6, advance fixed.Int26_6, ok bool) {
	glyph, ok := ts[r]
	// TODO: cache this, somehow?
	if !ok {
		return
	}
	intBounds := glyph.Bounds()
	return fixed.R(intBounds.Min.X, intBounds.Min.Y, intBounds.Max.X, intBounds.Max.Y), fixed.I(intBounds.Dx()), ok
}

func (ts TileSet) GlyphAdvance(r rune) (advance fixed.Int26_6, ok bool) {
	glyph, ok := ts[r]
	if !ok {
		return
	}
	return fixed.I(glyph.Bounds().Dx()), ok
}

func (ts TileSet) Kern(r0, r1 rune) fixed.Int26_6 {
	return fixed.I(0)
}

func (ts TileSet) Metrics() font.Metrics {
	return font.Metrics{
		// Height is the recommended amount of vertical space between two lines of
		// text.
		Height: fixed.I(16),

		// Ascent is the distance from the top of a line to its baseline.
		Ascent: fixed.I(16),

		// Descent is the distance from the bottom of a line to its baseline. The
		// value is typically positive, even though a descender goes below the
		// baseline.
		Descent: fixed.I(0),

		// XHeight is the distance from the top of non-ascending lowercase letters
		// to the baseline.
		XHeight: fixed.I(16), // not sure here

		// CapHeight is the distance from the top of uppercase letters to the
		// baseline.
		CapHeight: fixed.I(16), // not sure here

		// CaretSlope is the slope of a caret as a vector with the Y axis pointing up.
		// The slope {0, 1} is the vertical caret.
		CaretSlope: image.Point{0, 1},
	}
}

func FixedToImagePoint(fp fixed.Point26_6) image.Point {
	return image.Pt(fp.X.Round(), fp.Y.Round())
}

// fixed.Rectangle26_6 to image.Image rectangle
func FixedToImageRect(fr fixed.Rectangle26_6) image.Rectangle {
	return image.Rect(fr.Min.X.Round(), fr.Min.Y.Round(), fr.Max.X.Round(), fr.Max.Y.Round())
}
