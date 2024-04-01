package x3270

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"slices"
	"testing"

	"github.com/sparques/fansiterm/tiles"
)

func Test_Dumpx3270(t *testing.T) {
	dumpFontTileSet(Regular8x16)
}

func dumpFontTileSet(fts *tiles.FontTileSet) {
	glyphcount := len(fts.Glyphs)

	rows := glyphcount/30 + 1

	dst := image.NewRGBA(image.Rect(0, 0, 30*fts.Rectangle.Dx(), rows*fts.Rectangle.Dy()))
	pt := image.Pt(0, 0)

	runesPresent := make([]rune, len(fts.Glyphs))
	var i int
	for r := range fts.Glyphs {
		runesPresent[i] = r
		i++
	}

	slices.Sort(runesPresent)

	for _, r := range runesPresent {
		fts.DrawTile(r, dst, pt, color.White, color.Black)
		pt.X += fts.Rectangle.Dx()
		if pt.X > 30*fts.Rectangle.Dx() {
			pt.Y += fts.Rectangle.Dy()
			pt.X = 0
		}
	}

	save("dump.png", dst)
}

func save(fname string, img image.Image) {
	fh, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	png.Encode(fh, img)
	fh.Close()
}
