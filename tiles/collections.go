package tiles

import (
	"image"
	"image/color"
	"image/draw"
	"io"
	"os"
	"slices"
)

// MultiTileSet aggregates a set of Tilers together so that if one Tiler is
// missing a glyph, the next Tiler is used and so on.
type MultiTileSet struct {
	sets    []Tiler
	reverse bool
}

func NewMultiTileSet(sets ...Tiler) *MultiTileSet {
	return &MultiTileSet{sets: sets}
}

func (mts *MultiTileSet) GetTile(r rune) (image.Image, bool) {
	_, tile, ok := mts.lookupTile(r)
	return tile, ok
}

func (mts *MultiTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	ts, tile, ok := mts.lookupTile(r)
	if !ok {
		drawTile(dst, pt, EmptyTile, fg, bg)
		return
	}

	if drawer, ok := ts.(tileImageDrawer); ok {
		drawer.drawTileImage(tile, dst, pt, fg, bg)
		return
	}

	drawTile(dst, pt, tile, fg, bg)
}

func (mts *MultiTileSet) Prepend(t Tiler) {
	mts.sets = append(mts.sets, nil)
	copy(mts.sets[1:], mts.sets[:len(mts.sets)-1])
	mts.sets[0] = t
}

func (mts *MultiTileSet) Append(t Tiler) {
	mts.sets = append(mts.sets, t)
}

func (mts *MultiTileSet) Reverse(r bool) {
	if mts.reverse == r {
		return
	}
	slices.Reverse(mts.sets)
	mts.reverse = r
}

func (mts *MultiTileSet) lookupTile(r rune) (Tiler, image.Image, bool) {
	for _, ts := range mts.sets {
		tile, ok := ts.GetTile(r)
		if ok {
			return ts, tile, true
		}
	}
	return nil, nil, false
}

// Remap lets you remap runes in a Tiler.
type Remap struct {
	tileSet Tiler
	Map     map[rune]rune
}

func NewRemap(base Tiler) *Remap {
	return &Remap{
		tileSet: base,
		Map:     make(map[rune]rune),
	}
}

func (remap *Remap) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	if newr, ok := remap.Map[r]; ok {
		r = newr
	}
	remap.tileSet.DrawTile(r, dst, pt, fg, bg)
}

func (remap *Remap) GetTile(r rune) (image.Image, bool) {
	if newr, ok := remap.Map[r]; ok {
		r = newr
	}
	return remap.tileSet.GetTile(r)
}

type TileSet map[rune]image.Image

func NewTileSet() TileSet {
	return make(TileSet)
}

func (ts TileSet) GetTile(r rune) (image.Image, bool) {
	tile, ok := ts[r]
	return tile, ok
}

func (ts TileSet) SetTileFromFile(r rune, file string) error {
	fh, err := os.Open(file)
	if err != nil {
		return err
	}
	defer fh.Close()
	return ts.SetTileFromReader(r, fh)
}

func (ts TileSet) LoadTileFromFile(r rune, file string) {
	if err := ts.SetTileFromFile(r, file); err != nil {
		panic(err)
	}
}

func (ts TileSet) SetTileFromReader(r rune, rd io.Reader) error {
	img, _, err := image.Decode(rd)
	if err != nil {
		return err
	}
	ts[r] = img
	return nil
}

func (ts TileSet) LoadTileFromReader(r rune, rd io.Reader) {
	if err := ts.SetTileFromReader(r, rd); err != nil {
		panic(err)
	}
}

func (ts TileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	tile, ok := ts[r]
	if !ok {
		drawTile(dst, pt, EmptyTile, fg, bg)
		return
	}
	drawTile(dst, pt, tile, fg, bg)
}

func (ts TileSet) drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	drawTile(dst, pt, img, fg, bg)
}

type FullColorTileSet TileSet

func NewFullColorTileSet() FullColorTileSet {
	return make(FullColorTileSet)
}

func (fc FullColorTileSet) GetTile(r rune) (image.Image, bool) {
	tile, ok := fc[r]
	return tile, ok
}

func (fc FullColorTileSet) DrawTile(r rune, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	src, ok := fc[r]
	if !ok {
		return
	}
	drawFullColorTile(dst, pt, src, bg)
}

func (fc FullColorTileSet) drawTileImage(img image.Image, dst draw.Image, pt image.Point, fg color.Color, bg color.Color) {
	drawFullColorTile(dst, pt, img, bg)
}

func drawFullColorTile(dst draw.Image, pt image.Point, src image.Image, bg color.Color) {
	if rgba, ok := dst.(*image.RGBA); ok {
		drawFullColorTileRGBA(rgba, pt, src, bg)
		return
	}

	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	bgr, bgg, bgb, _ := bg.RGBA()

	for y := 0; y < height; y++ {
		srcY := bounds.Min.Y + y
		for x := 0; x < width; x++ {
			srcX := bounds.Min.X + x
			r, g, b, alpha := src.At(srcX, srcY).RGBA()
			switch alpha {
			case 0x00:
				dst.Set(pt.X+x, pt.Y+y, bg)
			case m:
				dst.Set(pt.X+x, pt.Y+y, color.RGBA{
					R: uint8(r >> 8),
					G: uint8(g >> 8),
					B: uint8(b >> 8),
					A: 0xFF,
				})
			default:
				dst.Set(pt.X+x, pt.Y+y, color.RGBA{
					R: alphaBlend(bgr, r, alpha),
					G: alphaBlend(bgg, g, alpha),
					B: alphaBlend(bgb, b, alpha),
					A: 0xFF,
				})
			}
		}
	}
}

func drawFullColorTileRGBA(dst *image.RGBA, pt image.Point, src image.Image, bg color.Color) {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	bgc := colorToRGBA8(bg)

	for y := 0; y < height; y++ {
		srcY := bounds.Min.Y + y
		dstRow := dst.PixOffset(pt.X, pt.Y+y)
		for x := 0; x < width; x++ {
			srcX := bounds.Min.X + x
			i := dstRow + x*4
			r, g, b, alpha := src.At(srcX, srcY).RGBA()
			switch alpha {
			case 0x0000:
				dst.Pix[i+0] = bgc.r
				dst.Pix[i+1] = bgc.g
				dst.Pix[i+2] = bgc.b
				dst.Pix[i+3] = bgc.a
			case m:
				dst.Pix[i+0] = uint8(r >> 8)
				dst.Pix[i+1] = uint8(g >> 8)
				dst.Pix[i+2] = uint8(b >> 8)
				dst.Pix[i+3] = 0xFF
			default:
				blended := blendRGBA8(bgc, rgba8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), 0xFF}, uint8(alpha>>8))
				dst.Pix[i+0] = blended.r
				dst.Pix[i+1] = blended.g
				dst.Pix[i+2] = blended.b
				dst.Pix[i+3] = blended.a
			}
		}
	}
}
